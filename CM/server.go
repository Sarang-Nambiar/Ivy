package CM

import (
	"fmt"
	"ivy/message"
	"ivy/utils"
	"net"
	"net/rpc"
	"os"
	"sync"
	"time"
)

type CentralManager struct {
	IP      string
	Records map[int]Record // Map of page id to record
	WriteQueue []WriteRequest
	IsBackup bool // To check if this is a backup central manager
	isDead bool // To check if the primary central manager is down
	IsRebooting bool // Boolean to represent if the central manager is rebooting
	Lock sync.Mutex
}

type WriteRequest struct {
	From Pointer
	PageID int
}

type Record struct {
	Copies  []Pointer
	Owner   Pointer
}

type Pointer struct {
	ID int
	IP string
}

const (
	READ = "READ"
	WRITE = "WRITE"
	PING = "PING"
	READ_FORWARD  = "READ_FORWARD"
	WRITE_FORWARD = "WRITE_FORWARD"
	RECEIVE_PAGE = "RECEIVE_PAGE"
	INVALIDATE_CONFIRMATION = "INVALIDATE_CONFIRMATION"
	WRITE_CONFIRMATION = "WRITE_CONFIRMATION"
	READ_CONFIRMATION = "READ_CONFIRMATION"
	INVALIDATE_CACHE = "INVALIDATE_CACHE"
	ACK = "ACK"
	LOCALHOST = "127.0.0.1:"
	CENTRALIP = LOCALHOST + "8000"
	BACKUPIP  = LOCALHOST + "8001"
	backupTime = 5 // Time interval for backing up the central manager metadata
	healthCheckTime = 5 // Time interval for health check of the primary central manager
)

// TODOS: Implement case when the primary central manager goes for rebooting
// Workflow for the above: make a new variable called primary central down
// This variable will be set to true when the primary central manager goes down
// The backup central during health check will see if the primary is back alive
// If the primary is back alive, then the backup central manager will sync its metadata with the primary
// Primary call Declare func to declare its the primary central manager

 // Function to start the RPC server
 func (cm *CentralManager) StartRPCServer() {
	rpc.Register(cm)

	listener, err := net.Listen("tcp", cm.IP)
	if err != nil {
		fmt.Printf("[CENTRAL-MANAGER] could not start listening: %s\n", err)
		os.Exit(1)
	}
	defer listener.Close()

	fmt.Printf("[CENTRAL-MANAGER] Node is running on %s\n", cm.IP)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("[CENTRAL-MANAGER] accept error: %s\n", err)
			continue
		}

		if cm.IsRebooting {
			conn.Close()
			time.Sleep(1 * time.Second)
			continue
		}

		go rpc.ServeConn(conn)
	}
}

func (cm *CentralManager) ReceiveRequest(msg message.Message, reply *message.Message) error {
	val, ok := cm.Records[msg.PageID]
	switch msg.Type {
	case PING: 
		// fmt.Printf("[CENTRAL-MANAGER] Received PING from client %d\n", msg.ID)
		*reply = message.Message{
			Type: ACK,
		}
	case READ:
		// fmt.Printf("[CENTRAL-MANAGER] Received READ for page %d from client %d\n", msg.PageID, msg.ID)
		if ok {
			// Page found in one of the clients
			// Forward the read request to the owner of the page
			msg.Type = READ_FORWARD
			val.Copies = append(val.Copies, Pointer{ID: msg.ID, IP: msg.IP})
			cm.Records[msg.PageID] = val
			// fmt.Printf("[CENTRAL-MANAGER] Forwarding READ request for page %d to client %d. Copies: %v\n", msg.PageID, val.Owner.ID, val.Copies)
			_, err := utils.CallByRPC(val.Owner.IP, "Client.ReceiveRequest", msg)
			if err != nil {
				return fmt.Errorf("error occurred while calling the client: %s", err)
			}

		} else {
			// Page not found in any of the clients
			// fmt.Printf("[CENTRAL-MANAGER] Page %d not found in any of the clients\n", msg.PageID)
			return fmt.Errorf("page not found in any of the clients")
		}
	case WRITE:
		// Check if the page exists in records
		// if so, then forward the request to the owner of the page
		// invalidate the cache of the copies of this page
		// if the page does not exist, then create a new record for this page

		cm.Lock.Lock()
		defer cm.Lock.Unlock()
		if cm.WriteQueue == nil {
			cm.WriteQueue = []WriteRequest{}
		}

		// fmt.Printf("[CENTRAL-MANAGER] Received WRITE for page %d from client %d\n", msg.PageID, msg.ID)

		if len(cm.WriteQueue) > 0 {
			head := cm.WriteQueue[0]
			if head.From.ID != msg.ID {
				// if the current request is not from the head, then add to the queue
				cm.WriteQueue = append(cm.WriteQueue, WriteRequest{From: Pointer{ID: msg.ID, IP: msg.IP}, PageID: msg.PageID})
				// fmt.Printf("[CENTRAL-MANAGER] Added WRITE request to the queue. Queue: %v\n", cm.WriteQueue)
				return nil
			} else {
				// if the current request is from the head, then do the write operation
				go cm.WriteOP(msg)
			}
		} else {
			// if the queue is empty, add the current write request to the queue
			// and do the write operation
			cm.WriteQueue = append(cm.WriteQueue, WriteRequest{From: Pointer{ID: msg.ID, IP: msg.IP}, PageID: msg.PageID})
			// fmt.Printf("[CENTRAL-MANAGER] Added WRITE request to the queue. Queue: %v\n", cm.WriteQueue)
			go cm.WriteOP(msg)
		}
	case READ_CONFIRMATION:
		fmt.Printf("[CENTRAL-MANAGER] Received READ_CONFIRMATION for page %d from client %d\n", msg.PageID, msg.ID)

	case WRITE_CONFIRMATION:
		// Remove the head of the write queue
		// Check if there are any more in the queue, then do a write operation for the next one

		cm.Lock.Lock()
		defer cm.Lock.Unlock()
		fmt.Printf("[CENTRAL-MANAGER] Received WRITE_CONFIRMATION for page %d from client %d\n", msg.PageID, msg.ID)
		if len(cm.WriteQueue) > 0 && cm.WriteQueue[0].From.ID == msg.ID {
			cm.WriteQueue = cm.WriteQueue[1:] // Remove the first element from the queue
		}
		cm.Records[msg.PageID] = Record{ // Initialize the new owner of the page
			Copies: []Pointer{},
			Owner: Pointer{ID: msg.ID, IP: msg.IP},
		}

		if len(cm.WriteQueue) > 0 {
			// if there are more requests in the queue, then do the write operation for the next one
			next := cm.WriteQueue[0]
			go func() {
				_, err := utils.CallByRPC(cm.IP, "CentralManager.ReceiveRequest", message.Message{Type: WRITE, ID: next.From.ID, IP: next.From.IP, PageID: next.PageID})
				if err != nil {
					fmt.Printf("error occurred while calling the client: %s", err)
				}
			}()
		}

	case INVALIDATE_CONFIRMATION: // Received when the client has invalidated the cache
		// Forward the write request to the owner of the page
		// TODO: Come back to this later
		fmt.Printf("[CENTRAL-MANAGER] Received INVALIDATE_CONFIRMATION for page %d from client %d\n", msg.PageID, msg.ID)
	}

	return nil
}

// Function to perform write operation flow
func (cm *CentralManager) WriteOP(msg message.Message) {
	val, ok := cm.Records[msg.PageID]
	if ok {
		// Page found in one of the clients
		// Invalidate the cache of the copies of this page and make the prev owner send the current copy with write perms to new owner
		for _, copy := range val.Copies {
			msg.Type = INVALIDATE_CACHE
			// fmt.Printf("[CENTRAL-MANAGER] Forwarding INVALIDATE_CACHE request to client %d\n", copy.ID)
			_, err := utils.CallByRPC(copy.IP, "Client.ReceiveRequest", msg)
			if err != nil {
				fmt.Printf("error occurred while calling the client: %s", err)
			}
		}

		// Forward the write request to the owner of the page
		msg.Type = WRITE_FORWARD
		_, err := utils.CallByRPC(val.Owner.IP, "Client.ReceiveRequest", msg)
		if err != nil {
			fmt.Printf("error occurred while calling the client: %s", err)
		}

	} else {
		// Page not found in any of the records
		// Create a new record for this page
		// Make the owner of the page the one who is writing
		go func() {
			_, err := utils.CallByRPC(msg.IP, "Client.ReceiveRequest", message.Message{Type: RECEIVE_PAGE, PageID: msg.PageID, Permission: WRITE})
			if err != nil {
				fmt.Printf("error occurred while calling the client: %s", err)
			}
			cm.Records[msg.PageID] = Record{
				Copies: []Pointer{},
				Owner: Pointer{ID: msg.ID, IP: msg.IP},
			}
		}()
	}
}

// Function to check the health of the primary central manager
func (cm *CentralManager) HealthCheck() {
	for {
		// Make a ping RPC call to the primary central manager
		_, err := utils.CallByRPC(CENTRALIP, "CentralManager.Ping", message.Message{Type: PING})
		if err != nil && !cm.isDead {
			fmt.Printf("[CENTRAL-MANAGER] The primary central manager is down. Starting the backup central manager...\n")
			cm.isDead = true
			var reply message.Message
			cm.DeclareCM(message.Message{}, &reply) // Function to declare the backup central manager as the primary central manager
		} else if err == nil && cm.isDead {
			fmt.Printf("[CENTRAL-MANAGER] The primary central manager is back alive. Syncing the metadata with the primary central manager...\n")
			cm.isDead = false
			// Sync the metadata with the primary central manager
			msg := SyncMessage{
				Records: cm.Records,
				WriteQueue: cm.WriteQueue,
			}
			client, err := rpc.Dial("tcp", CENTRALIP)
			if err != nil {
				fmt.Printf("[CENTRAL-MANAGER] Error in dialing: %s", err)
			}
		
			var reply SyncMessage
			err = client.Call("CentralManager.Backup", msg, &reply)
			if err != nil {
				fmt.Printf("[CENTRAL-MANAGER] Error in calling %s: %s", "CentralManager.Backup", err)
			}
			client.Close()

			// Call declare function in the central manager node
			_, err = utils.CallByRPC(CENTRALIP, "CentralManager.DeclareCM", message.Message{})
			if err != nil {
				fmt.Printf("[CENTRAL-MANAGER] Error occurred while declaring the primary central manager: %s\n", err)
			}
		}
		time.Sleep(healthCheckTime * time.Second)
	}
}

// Function to declare as primary central manager, applies to primary and backup central managers
func (cm *CentralManager) DeclareCM(msg message.Message, reply *message.Message) error {
	fmt.Printf("[CENTRAL-MANAGER] Declaring this central manager as the primary central manager\n")
	nodesList := utils.ReadNodesList()	
	for _, ip := range nodesList {
		go func() {
			_, err := utils.CallByRPC(ip, "Client.UpdateServerIP", message.Message{IP: cm.IP})
			if err != nil {
				fmt.Printf("[CENTRAL-MANAGER] Error occurred while updating the server IP: %s\n", err)
			}
		}()
	}
	return nil
}

func (cm *CentralManager) StartBackup() {
	for {
		// Make an RPC call here to the backup central manager
		msg := SyncMessage{
			Records: cm.Records,
			WriteQueue: cm.WriteQueue,
		}
		client, err := rpc.Dial("tcp", BACKUPIP)
		if err != nil {
			fmt.Printf("[CENTRAL-MANAGER] Error in dialing: %s", err)
		}
	
		var reply SyncMessage
		err = client.Call("CentralManager.Backup", msg, &reply)
		if err != nil {
			fmt.Printf("[CENTRAL-MANAGER] Error in calling %s: %s", "CentralManager.Backup", err)
		}
		client.Close()

		time.Sleep(backupTime * time.Second)
	}
}

func (cm *CentralManager) Backup(msg SyncMessage, reply *SyncMessage) error {
	fmt.Printf("[CENTRAL-MANAGER] Received backup message from primary central manager\n")
	cm.Records = msg.Records
	cm.WriteQueue = msg.WriteQueue
	return nil
}

func (cm *CentralManager) Ping(msg message.Message, reply *message.Message) error {
	return nil
}