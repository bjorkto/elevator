package main

import (
		"fmt"
		"net"
		"os"
)

//Error handling
func checkError(err error){
	if err != nil {
		fmt.Fprintf(os.Stderr,"Error: %s", err.Error())
		os.Exit(1)
	}
}

//Reads message sent from client and prints to console
func handleClient(conn net.Conn){
	var buf [512]byte
	for{
		n, err := conn.Read(buf[0:])	//reads message sent by client (up to 512 bytes of data)
		if err != nil{
			return
		}
		fmt.Println(string(buf[0:n]))	//print the received message to console
	}
}


func main() {
	//Listen to port 1200
	service := ":1200"
	tcpAddr, err := net.ResolveTCPAddr("tcp4", service)
	checkError(err)
	listener, err := net.ListenTCP("tcp", tcpAddr)
	checkError(err)
	
	for{
		conn, err := listener.Accept()	//accept connection
		if err != nil {
			continue
		}
		
		handleClient(conn)		//handle the connection with the client in its own function
		conn.Close()			//close connection when done
	}
}