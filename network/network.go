package network

import (
		"fmt"
		"time"
		"net"
		"os"
		"strings"
)


//Error handling
func checkError(err error){
    if err != nil {
        fmt.Fprintf(os.Stderr,"Error: %s", err.Error())
        os.Exit(1)
    }
}

/*
-------------------------------------------
--------- Master functionallity -----------
-------------------------------------------
*/

//Broadcast on UDP to signal that a master exists
func BroadcastUDP(service string){
    
	fmt.Println("Broadcasting on UDP ", service)
    udpAddr, err := net.ResolveUDPAddr("udp4", service)
    checkError(err)
    conn, err := net.DialUDP("udp", nil, udpAddr)
    checkError(err)
		
	for{
		conn.Write([]byte("Stian og Vegards heiser"))
		time.Sleep(500*time.Millisecond)
    }
	
}


//Set up a TCP server and listen for connections
func StartTCPServer(port string, connChan chan *net.TCPConn){
	    
	tcpAddr, err := net.ResolveTCPAddr("tcp4", port)
    checkError(err)
    listener, err := net.ListenTCP("tcp", tcpAddr)
    checkError(err)
	
	fmt.Println("TCP server started, listening for connections...")

	//listen for inncoming connections and spawn new threads to handle each client
	for{
		conn, err := listener.AcceptTCP()		//accept connection
		if err != nil {
				continue
		}
		fmt.Println("Accepted connection from: ", conn.RemoteAddr())	//print address of the client
		
		//Send the connection pointer to the main thread
		connChan <- conn
		
		//spawn go-routine that reads messages from the client 
		go ListenToClient(conn)
	}
}


//Listen for messages from the client
func ListenToClient(conn *net.TCPConn){
	addr := conn.RemoteAddr()
	for{
		var buf [1024]byte
        n, err := conn.Read(buf[0:])		//read message
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		fmt.Println(addr, " says: ", string(buf[0:n])) 
		
    }
}


/*
-------------------------------------------
--------- Client functionallity -----------
-------------------------------------------
*/


//Check if master exists by listening for UDP message
func SearchForMaster(port string) (bool, string) {
	
    fmt.Println("Listening for master...")
    udpAddr, err := net.ResolveUDPAddr("udp4", port)
    checkError(err)
       
    listener, err := net.ListenUDP("udp", udpAddr)
    checkError(err)

	//timeout after 3 seconds
	listener.SetReadDeadline(time.Now().Add(3*time.Second))

	var buf [1024]byte
	master_exists := false	
	master_address := ""

	for !master_exists{
		n, addr, _ := listener.ReadFromUDP(buf[0:])	//read UDP message
	
		msg:=string(buf[0:n]);

		if(n==0){
			//timeout has happened
			break		
		}else if(msg=="Stian og Vegards heiser"){
			//master exists
			master_exists = true		
			full_address := fmt.Sprintf("%v", addr)	//convert master address to string	
			master_address = full_address[0:strings.Index(full_address, ":")]  //remove the udp port number
		}
	}
	return master_exists, master_address
}


//connect to master with TCP
func ConnectToMaster(masterAddr string) *net.TCPConn{
	
	service := masterAddr + ":10002"
	fmt.Println("Attemting to connect to master...", )
	tcpAddr, err := net.ResolveTCPAddr("tcp4", service)
	if (err != nil) {
		return nil
	}
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if (err != nil) {
		return nil
	}
	fmt.Println("Connection established!")
	return conn
}