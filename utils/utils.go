package utils

import (
	"fmt"
	"ivy/message"
	"net/rpc"
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