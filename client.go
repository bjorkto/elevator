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


func main(){
	//Connect to port 1200 on loacalhost
	tcpAddr, err := net.ResolveTCPAddr("tcp4", "localhost:1200")
	checkError(err)
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	checkError(err)
	
	loop := true
	for loop{
		reader := bufio.NewReader(os.Stdin)		//create a reader that reads from console input
		line, _ := reader.ReadString('\n')		//read entire line of input as string
		
		if line == string("exit\r\n") {			//exit if "exit" has been entered
			loop = false
		
		} else{
			//send data to the server
			_, err_write := conn.Write([]byte(line))		
			checkError(err_write)
			
			//recieve the response
			var buf [512]byte
			n, err_read := conn.Read(buf[0:])
			checkError(err_read)
			fmt.Println("Server response: ", string(buf[0:n]))
		}
	}
	
	conn.Close()	//close connection
}