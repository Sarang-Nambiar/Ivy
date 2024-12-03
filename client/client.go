package client

import (
	"fmt"
	"net"
	"net/rpc"
	"os"
)

type Client struct {
	ID     int
	IP     string
	Cached map[int]Page
}

type Pointer struct {
	ID int
	IP string
}

type Page struct {
	Num        int    // Page number
	Permission string // READ | WRITE
}

const (
	READ      = "READ"
	WRITE     = "WRITE"
	NUM_PAGES = 4
	LOCALHOST = "127.0.0.1:"
	CENTRALIP = LOCALHOST + "8000"
	BACKUPIP  = LOCALHOST + "8001"
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

func (c *Client) Join() error {
	// Implement central server ping check
	// If central is down, join backup

	return nil
}

func (c *Client) Read(pageID int) error {
	// Implement read from central
	return nil
}

func (c *Client) Write(pageID int, data string) error {
	// Implement write to central
	return nil
}
