package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"ivy/CM"
	"ivy/client"
	"ivy/message"
	"ivy/utils"
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

			for {
				var answer string
				fmt.Println("Make sure you have all the clients running before starting the read and write requests.")
				fmt.Println("Do you want to start the read and write requests from the clients? (y/n)")
				fmt.Scanln(&answer)
	
				if answer == "y" {
					nodesList := utils.ReadNodesList()
					fmt.Printf("Nodes list: %v\n", nodesList)
					for _, ip := range nodesList {
						go func() {
							_, err := utils.CallByRPC(ip, "Client.RequestPage", message.Message{})
							if err != nil {
								fmt.Println("Error occurred while calling RequestPage RPC: ", err)
							}
						}()
					}
					break
				} else {
					fmt.Println("The option to start the read and write requests will be displayed again shortly...")
				}
			}

			// for {
			// 	utils.ShowMenu()
			// 	var choice int 
			// 	fmt.Scanln(&choice)
			// 	switch choice {
			// 	case 1:
			// 		// Display the records
			// 		fmt.Printf("Records:\n")
			// 		for key, val := range cm.Records {
			// 			fmt.Printf("PageID: %d and Owner: %d\n", key, val.Owner.ID)
			// 		}
			// 	case 2:
			// 		// Display the write queue
			// 		fmt.Printf("Write Queue:\n")
			// 		for _, val := range cm.WriteQueue {
			// 			fmt.Printf("PageID: %d and Owner: %d\n", val.PageID, val.From.ID)
			// 		}
			// 	default:
			// 		fmt.Printf("Invalid choice: %d\n", choice)
			// 	}
			// }

			// Display options to show the records in the central manager

		case "-b":
			// Start the backup central manager

			cm := CM.CentralManager{
				IP: client.BACKUPIP,
				Records: make(map[int]CM.Record),
			}

			go cm.StartRPCServer()

		case "-cl":
			// Start the client
			nodesList := utils.ReadNodesList()
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
				nodesList := utils.ReadNodesList()

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