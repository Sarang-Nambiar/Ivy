package utils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"ivy/message"
	"net/rpc"
	"os"
)

// Utility function to call RPC methods
func CallByRPC(IP string, method string, msg message.Message) (message.Message, error) {
	client, err := rpc.Dial("tcp", IP)
	if err != nil {
		return message.Message{}, fmt.Errorf("error in dialing: %s", err)
	}
	defer client.Close()

	var reply message.Message
	err = client.Call(method, msg, &reply)
	if err != nil {
		return message.Message{}, fmt.Errorf("error in calling %s: %s", method, err)
	}
	return reply, nil
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

func ShowMenu(){
	red := "\033[31m"  // ANSI code for red text
	reset := "\033[0m" // ANSI code to reset color

	fmt.Println(red + "--------------------------------" + reset)
	fmt.Println(red + "\t\tMENU" + reset)
	fmt.Println(red + "Enter 1 to see the records" + reset)
	fmt.Println(red + "Enter 2 to see the Write queue" + reset)
	fmt.Println(red + "Enter 3 to kill current node" + reset)
	fmt.Println(red + "Enter 4 to reboot current node" + reset)
	fmt.Println(red + "--------------------------------" + reset)
}