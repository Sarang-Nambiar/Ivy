package CM

import (
	"fmt"
	"ivy/client"
	"ivy/message"
	"net"
	"net/rpc"
	"os"
)

type CentralManager struct {
	IP      string
	Records map[int]Record // Map of page id to record
}

type Record struct {
	Copies  []client.Pointer
	Owner   client.Pointer
}

const (
	ACK = "ACK"
)

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
		go rpc.ServeConn(conn)
	}
}


func (cm *CentralManager) Ping(msg message.Message, reply *message.Message) error {
	*reply = message.Message{
		Type: ACK,
	}
	return nil
}