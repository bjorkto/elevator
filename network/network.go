package network

import (
		"fmt"
		"time"
		"net"
		"os"
		"strings"
		"strconv"
		"os/exec"
)


var NetworkMode = bool(true)


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
func StartTCPServer(port string, newconnChan chan *net.TCPConn, delconnChan chan *net.TCPConn){
	    
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
		newconnChan <- conn
		
		//spawn go-routine that reads messages from the client 
		go sendhandshake(conn)
		go ListenToClient(conn, delconnChan)
	}
}


//Listen for messages from the client
func ListenToClient(conn *net.TCPConn, delconnChan chan *net.TCPConn){
	addr := conn.RemoteAddr()
	var buf [1024]byte
	for{
		//set timeout to two seconds
		conn.SetReadDeadline(time.Now().Add(2*time.Second))
		
		//read message
		n, err := conn.Read(buf[0:])
		
		//if error or timeout happens, assume we have lost connection with the client
		if (err != nil || n == 0) {
			fmt.Println("Lost connection with", conn.RemoteAddr())
			delconnChan <- conn
			conn.Close()
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
	listener.Close()
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
	
	go sendhandshake(conn)
	go ListenToMaster(conn)
	
	return conn
}


func StartNewMaster(){
	fmt.Println("Starting new master...")
	cmd := exec.Command("mate-terminal", "-x", "go", "run",  "Master.go")
	err := cmd.Start()
	if err != nil {
		fmt.Println(err.Error())
	}
}


//Handle a lost connection to the master
func handleLostConnection(masterQueue int){
	fmt.Println("Lost connection with Master")
	exist := false
	masteraddr := ""
	//try to find or create a new master, based on the place in the queue
	for !exist {
		if(masterQueue == 1){
			fmt.Println("I'm the new master")
			StartNewMaster()
			masterQueue = -1
		if(masterQueue < 1){
			fmt.Println("Something is very wrong... I cannot even connect to myself!")
			fmt.Println("Changing to non network mode")
			NetworkMode = false
		}
		}else{
			masterQueue -= 1
			fmt.Println("Searching for new master... becoming master in master", masterQueue, "tries")
		}
		exist, masteraddr = SearchForMaster(":10001")
	}
	//Connect to the new master
	ConnectToMaster(masteraddr)
}


//Listening to master for orders 
func ListenToMaster(conn *net.TCPConn){
	masterQueue := -1
	var buf [1024]byte
        
	for{
		
		//set timeout to two seconds
		conn.SetReadDeadline(time.Now().Add(2*time.Second))
		
		//read message
		n, err := conn.Read(buf[0:])
		
		//if error or timeout happens, assume we have lost connection with the master
		if (err != nil || n == 0) {
			conn.Close()
			handleLostConnection(masterQueue)
			return
		}
		
		msgType := buf[0]
        switch msgType{
			case 0:
				//type 0 is only a handshake. Nothing to handle. 
				break
			case 1:
				//type 1 is the place in the queue to become the new master if anything goes wrong
				masterQueue, _ = strconv.Atoi(string(buf[1:n]))
				fmt.Println(masterQueue)
				break
			case 2:
				//type 2 is a jobOrder.
				break
			case 3:
				//type 3 is a backup
		}
    }
}


/*
-------------------------------------------
--------- Shared functionallity -----------
-------------------------------------------
*/


//send a handshake to signal that the process is still alive and connected
func sendhandshake(conn *net.TCPConn){
	for{
		_, err := conn.Write([]byte("Handshake"))
		if err != nil {
			return
		}
	}
}

//just print the error message
func checkError(err error){
    if err != nil {
        fmt.Fprintf(os.Stderr,"Error: %s", err.Error())
        os.Exit(1)
    }
}


