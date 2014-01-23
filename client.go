package main

import (
		"fmt"
		"net"
		"os"
		"bufio"
)

//Error handling
func checkError(err error){
	if err != nil {
		fmt.Fprintf(os.Stderr,"Error: %s", err.Error()) 
		os.Exit(1)
	}
}

//Read messages from server and print to console
func listenToServer(conn net.Conn){
	for{
		var buf [512]byte
		n, err_read := conn.Read(buf[0:])
		checkError(err_read)
		fmt.Println("Server says: ", string(buf[0:n]))
	}
}


func main(){

	//Connect to port 1200 on loacalhost
	service := "localhost:1200"
	fmt.Println("Attemting to connect to ", service)
	tcpAddr, err := net.ResolveTCPAddr("tcp4", service)
	checkError(err)
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	checkError(err)
	
	go listenToServer(conn)		//Thread for receiving messages from the server
	
	loop := true
	for loop{
		reader := bufio.NewReader(os.Stdin)		//create a reader that reads from console input
		line, _ := reader.ReadString('\n')		//read entire line of input as string
		
		if line == string("exit\r\n") {			//exit if "exit" has been entered
			loop = false
		
		} else{
			_, err := conn.Write([]byte(line))	//send data to the server	
			checkError(err)
		}
	}
	conn.Close()	//close connection
}