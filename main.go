package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"ivy/CM"
	"ivy/client"
	"os"
	"os/signal"
	"syscall"
)

func main() {

	args := os.Args

	if len(args) < 2 {
		fmt.Println("Usage: go run main.go -cm or go run main.go -cl or go run main.go -b")
		return
	}

	switch args[1] {
		case "-cm":
			// Start the central manager
			cm := CM.CentralManager{
				IP: client.CENTRALIP,
				Records: make(map[int]CM.Record),
			}

			// Start the RPC server
			go cm.StartRPCServer()
		case "-b":
			// Start the backup central manager

			cm := CM.CentralManager{
				IP: client.BACKUPIP,
				Records: make(map[int]CM.Record),
			}

			go cm.StartRPCServer()

		case "-cl":
			// Start the client
			nodesList := ReadNodesList()
			client := client.Client{
				ID: len(nodesList),
				IP: client.LOCALHOST + fmt.Sprint(8002+len(nodesList)),
				Cached: make(map[int]client.Page),
			}
			
			nodesList[client.ID] = client.IP

			jsonData, err := json.Marshal(nodesList)
			if err != nil {
				fmt.Println("Error occurred while marshalling the Ring into nodes-list.json: ", err)
			}
			
			err = ioutil.WriteFile("nodes-list.json", jsonData, os.ModePerm)
			if err != nil {
				fmt.Println("Error occurred while marshalling the Ring back into nodes-list.json: ", err)
			}

			go client.StartRPCServer()

			// Handling when the node fails or is shut down
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

			// For cleanup after the node is shut down
			go func() {
				<-sigChan
				fmt.Println("Shutting down...")

				// Remove the node from the list
				nodesList := ReadNodesList()

				delete(nodesList, client.ID) // remove the element that left the network from the nodesList

				jsonData, err := json.Marshal(nodesList)
				if err != nil {
					fmt.Println("Error occurred while updating nodes-list.json: ", err)
				}
				err = ioutil.WriteFile("nodes-list.json", jsonData, os.ModePerm)
				if err != nil {
					fmt.Println("Error occurred while updating nodes-list.json: ", err)
				}
				os.Exit(0)
			}()
		default:
			fmt.Printf("Invalid argument: %s\n", args[1])
			return
		}
	select {}
}

func ReadNodesList() map[int]string {
	jsonFile, err := os.Open("nodes-list.json")
	if err != nil {
		fmt.Println("Error opening nodes-list.json file:", err)
	}
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	var nodesList map[int]string

	json.Unmarshal(byteValue, &nodesList) // Puts the byte value into the nodesList map

	return nodesList
}