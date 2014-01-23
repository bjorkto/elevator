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

//Handles messages from the client
func handleClient(conn net.Conn){
	
	//print welcome message
	_, err := conn.Write([]byte("Welcome to the echo server!\n"))
	if err != nil{
		return
	}
	
	var buf [512]byte
	for{
		n, err_read := conn.Read(buf[0:])	//reads message sent by client (up to 512 bytes of data)
		if err_read != nil{
			return
		}
		
		fmt.Println(conn.RemoteAddr(), "says: ", string(buf[0:n]))			//print the received message to console
		
		_, err_write:= conn.Write(buf[0:n])		//echo the signal back to the client
		if err_write != nil{
			return
		}
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
		fmt.Println("Accepted connection from: ", conn.RemoteAddr())
		go handleClient(conn)		//handle the connection with the client in its own function
	}
}