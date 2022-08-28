package main

import (
	"fmt"
	"net"
	"os"
)

func main() {
	serverEP := "192.168.1.21:1162"

	conn, err := net.Dial("udp", serverEP)
	if err != nil {
		fmt.Printf("Dial err %v", err)
		os.Exit(-1)
	}
	defer conn.Close()

	msg := "1234567890"
	fmt.Printf("Ping: %v\n", msg)
	if _, err = conn.Write([]byte(msg)); err != nil {
		fmt.Printf("Write err %v", err)
		os.Exit(-1)
	}

}
