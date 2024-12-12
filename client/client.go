package client

import (
	"fmt"
	"ivy/message"
	"ivy/utils"
	"math/rand"
	"net"
	"net/rpc"
	"os"
	"sync"
	"time"
)

var (
	totalReadTime float64
	totalWriteTime float64
	count map[string]int = make(map[string]int)
)

type Client struct {
	ID     int
	IP     string
	Cached map[int]Page
	StartTime time.Time
	ServerIP string
	Lock sync.Mutex
}

type Page struct {
	ID        int    // Page number
	Permission string // READ | WRITE
}

const (
	READ      = "READ"
	WRITE     = "WRITE"
	PING	  = "PING"
	ACK	   = "ACK"
	RECEIVE_PAGE = "RECEIVE_PAGE"
	READ_FORWARD = "READ_FORWARD"
	WRITE_FORWARD = "WRITE_FORWARD"
	WRITE_CONFIRMATION = "WRITE_CONFIRMATION"
	READ_CONFIRMATION = "READ_CONFIRMATION"
	INVALIDATE_CACHE = "INVALIDATE_CACHE"
	INVALIDATE_CONFIRMATION = "INVALIDATE_CONFIRMATION"
	NUM_PAGES = 4 // Number of pages in the system
	LOCALHOST = "127.0.0.1:"
	CENTRALIP = LOCALHOST + "8000"
	BACKUPIP  = LOCALHOST + "8001"
	NUMREQUESTS = 10
	RED = "\033[31m"  // ANSI code for red text
	RESET = "\033[0m" // ANSI code to reset color
	timeInterval = 10 // Time interval between requests
)

// Function to start the RPC server
func (c *Client) StartRPCServer() {
	rpc.Register(c)

	listener, err := net.Listen("tcp", c.IP)
	if err != nil {
		fmt.Printf("[NODE-%d] could not start listening: %s\n", c.ID, err)
		os.Exit(1)
	}
	defer listener.Close()

	fmt.Printf("[NODE-%d] Node is running on %s\n", c.ID, c.IP)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("[NODE-%d] accept error: %s\n", c.ID, err)
			continue
		}
		go rpc.ServeConn(conn)
	}
}

// Function to request for read/write access for a page from the central server
func (c *Client)RequestPage(msg message.Message, reply *message.Message) error {
	for i := 0; i < NUMREQUESTS; i++ {

		// coin flip to choose read or write request
		// requestType := c.coinFlip() // randomized flip
		// requestType := c.percentageBasedFlip(90) // 90% read and 10% write(read-intensive workload)
		requestType := c.percentageBasedFlip(10) // 10% read and 90% write(write-intensive workload)
		// Randomly choose a page to read/write
		pageID := rand.Intn(NUM_PAGES)

		c.StartTime = time.Now()
		switch requestType {
			case READ:
				fmt.Printf("[NODE-%d] Requesting READ access for page %d\n", c.ID, pageID)
				if val, ok := c.Cached[pageID]; ok {
					fmt.Printf("[NODE-%d] Page %d is already in the cache with permission %s\n", c.ID, pageID, val.Permission)
					break
				}

				_, err := utils.CallByRPC(c.ServerIP, "CentralManager.ReceiveRequest", message.Message{Type: READ, ID: c.ID, IP: c.IP, PageID: pageID})
				if err != nil {
					fmt.Printf("[NODE-%d] Error occurred while requesting READ access for page %d: %s\n", c.ID, pageID, err)
				}

			case WRITE:
				fmt.Printf("[NODE-%d] Requesting WRITE access for page %d\n", c.ID, pageID)
				if val, ok := c.Cached[pageID]; ok && val.Permission == WRITE {
					fmt.Printf("[NODE-%d] Page %d is already in the cache with permission %s\n", c.ID, pageID, val.Permission)
					break
				}
				
				_, err := utils.CallByRPC(c.ServerIP, "CentralManager.ReceiveRequest", message.Message{Type: WRITE, ID: c.ID, IP: c.IP, PageID: pageID})
				if err != nil {
					fmt.Printf("[NODE-%d] Error occurred while requesting WRITE access for page %d: %s\n", c.ID, pageID, err)
				}
		}
		time.Sleep(timeInterval * time.Second)
	}
	// Exit the program once all the requests are done
	fmt.Printf("[NODE-%d] All requests are done\n", c.ID)
	var avgReadTime float64 = 0
	var avgWriteTime float64 = 0
	if count[READ] != 0 {
		avgReadTime = totalReadTime/float64(count[READ])
	}
	if count[WRITE] != 0 {
		avgWriteTime = totalWriteTime/float64(count[WRITE])
	}
	_, err := utils.CallByRPC(c.ServerIP, "CentralManager.CalculateAverageResponseTime", message.Message{AvgReadPerNode: avgReadTime, AvgWritePerNode: avgWriteTime})
	if err != nil {
		fmt.Printf("[NODE-%d] Error occurred while sending the average read and write time to the central manager: %s\n", c.ID, err)
	}
	return nil
}

func (c *Client) ReceiveRequest(msg message.Message, reply *message.Message) error {
	switch msg.Type {
	case RECEIVE_PAGE:
		fmt.Printf("[NODE-%d] Received page %d with permission %s\n", c.ID, msg.PageID, msg.Permission)
		c.Lock.Lock()
		c.Cached[msg.PageID] = Page{ID: msg.PageID, Permission: msg.Permission}
		fmt.Printf("[NODE-%d] Updated cache: %v\n", c.ID, c.Cached)
		c.Lock.Unlock()

		if msg.Permission == WRITE {
			// Send the confirmation to the central manager
			_, err := utils.CallByRPC(c.ServerIP, "CentralManager.ReceiveRequest", message.Message{Type: WRITE_CONFIRMATION, ID: c.ID, IP: c.IP, PageID: msg.PageID})
			if err != nil {
				return fmt.Errorf("error occurred while calling the central manager: %s", err)
			}
			totalWriteTime += float64(time.Since(c.StartTime).Seconds())
		} else {
			// Send the confirmation to the central manager
			_, err := utils.CallByRPC(c.ServerIP, "CentralManager.ReceiveRequest", message.Message{Type: READ_CONFIRMATION, ID: c.ID, IP: c.IP, PageID: msg.PageID})
			if err != nil {
				return fmt.Errorf("error occurred while calling the central manager: %s", err)
			}
			totalReadTime += float64(time.Since(c.StartTime).Seconds())
		}

		if _, ok := count[msg.Permission]; !ok {
			count[msg.Permission] = 0
		}

		count[msg.Permission]++
		fmt.Printf(RED + "[NODE-%d] Total time taken for the request: %v\n" + RESET, c.ID, time.Since(c.StartTime))

	case READ_FORWARD:
		// Forward the read request to the client
		fmt.Printf("[NODE-%d] Forwarding READ permission for page %d to the client %d\n", c.ID, msg.PageID, msg.ID)
		msg.Type = RECEIVE_PAGE
		msg.Permission = READ

		c.Lock.Lock()
		c.Cached[msg.PageID] = Page{ID: msg.PageID, Permission: READ} // Making sure that the perms for that page is set to READ
		c.Lock.Unlock()

		_, err := utils.CallByRPC(msg.IP, "Client.ReceiveRequest", msg)
		if err != nil {
			return fmt.Errorf("error occurred while calling the client: %s", err)
		}

	case WRITE_FORWARD:
		// Forward the write request to the client
		fmt.Printf("[NODE-%d] Forwarding WRITE permission for page %d to the client %d\n", c.ID, msg.PageID, msg.ID)
		msg.Type = RECEIVE_PAGE
		msg.Permission = WRITE

		_, err := utils.CallByRPC(msg.IP, "Client.ReceiveRequest", msg)
		if err != nil {
			fmt.Printf("error occurred while calling the client: %s\n", err)
		}

		c.Lock.Lock()
		delete(c.Cached, msg.PageID) // Invalidate the cache from the owner
		c.Lock.Unlock()

		fmt.Printf("[NODE-%d] Updated cache: %v\n", c.ID, c.Cached)

	case INVALIDATE_CACHE:
		// Invalidate the cache
		fmt.Printf("[NODE-%d] Invalidating the cache for page %d\n", c.ID, msg.PageID)

		c.Lock.Lock()
		delete(c.Cached, msg.PageID) // removed the cached page from the client
		c.Lock.Unlock()
		
		// Send the confirmation to the central manager
		_, err := utils.CallByRPC(c.ServerIP, "CentralManager.ReceiveRequest", message.Message{Type: INVALIDATE_CONFIRMATION, ID: c.ID, IP: c.IP, PageID: msg.PageID})
		if err != nil {
			return fmt.Errorf("error occurred while calling the central manager: %s", err)
		}
	}
	return nil
}

// Updating the server IP one of the CMs are down
func (c *Client) UpdateServerIP(msg message.Message, reply *message.Message) error {
	c.ServerIP = msg.IP
	return nil
}

// coin flip to choose read or write request
func (c *Client) coinFlip() string{
	rand.Seed(time.Now().UnixNano()) // Making sure this is random using a unique seed
	outcome := rand.Intn(2)
	if outcome == 0 {
		return READ
	}
	return WRITE
}

// For Read-intensive and Write-intensive workloads
func (c *Client) percentageBasedFlip(readPercentage int) string {
	rand.Seed(time.Now().UnixNano())
	outcome := rand.Intn(100)
	if outcome < readPercentage {
		return READ
	}
	return WRITE
}
