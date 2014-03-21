/*This is the network module. It contains the network commucation functionality*/

package network

import (
	. "../datatypes"
	. "../encoder"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"
)



/*
-------------------------------------------
--------- Master functionallity -----------
-------------------------------------------
*/

//Broadcast on UDP to signal that a master exists
func BroadcastUDP(service string) {

	fmt.Println("Broadcasting on UDP ", service)
	udpAddr, err := net.ResolveUDPAddr("udp4", service)
	checkError(err)
	conn, err := net.DialUDP("udp", nil, udpAddr)
	checkError(err)

	for {
		conn.Write([]byte("Stian og Vegards heiser"))
		time.Sleep(200 * time.Millisecond)
	}

}

//Set up a TCP server and listen for connections
func StartTCPServer(port string, newconnChan chan *net.TCPConn, lostConnChan chan *net.TCPConn, msgChan chan Message) {

	tcpAddr, err := net.ResolveTCPAddr("tcp4", port)
	checkError(err)
	listener, err := net.ListenTCP("tcp", tcpAddr)
	checkError(err)

	fmt.Println("TCP server started, listening for connections...")

	//listen for inncoming connections and spawn new threads to handle each client
	for {
		conn, err := listener.AcceptTCP() //accept connection
		if err != nil {
			continue
		}
		fmt.Println("Accepted connection from: ", conn.RemoteAddr()) //print address of the client

		//Send the connection pointer to the main thread
		newconnChan <- conn

		//spawn go-routines that reads messages from the client and sends handshakes
		go sendhandshake(conn)
		go ListenForMessages(conn, lostConnChan, msgChan)
	}
}

/*
-------------------------------------------
---------- Slave functionallity -----------
-------------------------------------------
*/

//Check if master exists by listening for UDP message
func SearchForMaster(port string) (bool, string) {
	udpAddr, err := net.ResolveUDPAddr("udp4", port)
	checkError(err)

	listener, err := net.ListenUDP("udp", udpAddr)
	checkError(err)

	//timeout after 1 second
	listener.SetReadDeadline(time.Now().Add(1 * time.Second))

	var buf [1024]byte
	master_exists := false
	master_address := ""

	for !master_exists {

		n, addr, _ := listener.ReadFromUDP(buf[0:]) //read UDP message

		msg := string(buf[0:n])

		if n == 0 {
			//timeout has happened
			break
		} else if msg == "Stian og Vegards heiser" {
			//master exists
			master_exists = true
			full_address := fmt.Sprintf("%v", addr)                           //convert master address to string
			master_address = full_address[0:strings.Index(full_address, ":")] //remove the udp port number
		}
	}
	listener.Close()
	return master_exists, master_address
}

//connect to master with TCP
func ConnectToMaster(masterAddr string, msgChan chan Message, lostMasterChan chan *net.TCPConn) *net.TCPConn {

	service := masterAddr + ":10002"
	fmt.Println("Attemting to connect to master...")
	tcpAddr, err := net.ResolveTCPAddr("tcp4", service)
	if err != nil {
		return nil
	}
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return nil
	}
	
	//start threads that reads messages from master and sends handshakes
	go ListenForMessages(conn, lostMasterChan, msgChan)
	go sendhandshake(conn)
	
	fmt.Println("Connection established!")
	return conn
}

//Start a new master process
func StartNewMaster() {
	fmt.Println("Starting new master...")
	cmd := exec.Command("mate-terminal", "-x", "go", "run", "Master.go")
	err := cmd.Start()
	if err != nil {
		fmt.Println(err.Error())
	}
}

//Handle a lost connection to the master
func HandleLostConnection(masterQueue int, msgChan chan Message, lostMasterChan chan *net.TCPConn, newMasterChan chan *net.TCPConn) {
	exist := false
	masteraddr := ""
	//try to find or create a new master, based on the place in the queue
	for !exist {
		if masterQueue == 1 {
			fmt.Println("I'm the new master")
			StartNewMaster()
			masterQueue -= 1
		} else if masterQueue == 0 {
			fmt.Println("Something is very wrong... I cannot even connect to myself! I will keep trying to connect in the background...")
			cmd := exec.Command("mate-terminal", "-x", "killall",  "Master")
			err := cmd.Start()
			if err != nil {
				fmt.Println(err.Error())
			}
			masterQueue -= 1
		} else if masterQueue > 1 {
			masterQueue -= 1
			fmt.Println("Searching for new master... becoming master in master", masterQueue, "tries")
		}
		exist, masteraddr = SearchForMaster(":10001")
	}
	//Connect to the new master
	conn := ConnectToMaster(masteraddr, msgChan, lostMasterChan)
	newMasterChan <- conn
}

/*
-------------------------------------------
--------- Shared functionallity -----------
-------------------------------------------
*/

//send a handshake to signal that the process is still alive and connected
func sendhandshake(conn *net.TCPConn) {
	if (conn != nil) {
		for {
			time.Sleep(500 * time.Millisecond)
			 _, err := conn.Write([]byte("0Handshake"))
			if err != nil {
				return
			}
		}
	}
}

//Listen for messages over the TCP connection
func ListenForMessages(conn *net.TCPConn, lostConnChan chan *net.TCPConn, msgChan chan Message) {
	addr := conn.RemoteAddr()
	var buf [1024]byte
	for {
		//set timeout to one second
		conn.SetReadDeadline(time.Now().Add(1 * time.Second))

		//read message
		n, err := conn.Read(buf[0:])

		//if error or timeout happens, assume we have lost connection
		if err != nil || n == 0 {
			fmt.Println("Lost connection with", addr)
			lostConnChan <- conn
			conn.Close()
			return
		}
		
		//if sending messages to fast, the buffer may conatin several messages. Split them!
		str := string(buf[0:n])
		messages := strings.SplitAfter(str, "\n")
		for i := 0; i < len(messages)-1; i++ {
			//send message to the thread that handles it
			var m Message
			m.Sender = conn
			m.Msg = messages[i][0:len(messages[i])-1]
			msgChan <- m
		}
	}
}

//send data over the TCP connection
//data can theoretically be of any type, find out wich by using switch on the type
func SendMessage(conn *net.TCPConn, data interface{}) {
	switch data.(type) {
	case int:
		//data is a queue number
		data := data.(int)
		msg := EncodeQueue(data)
		if (conn != nil) {
	        conn.Write([]byte(msg))
        }
		break
	case Event:
		//data is a button event
		data := data.(Event)
		msg := EncodeEvent(data)
		if (conn != nil) {
	        conn.Write([]byte(msg))
        }
		break
	case ElevatorStruct:
		//data is an elevatorStruct
		data := data.(ElevatorStruct)
		msg := EncodeElevatorStruct(data)
		if (conn != nil) {
	        conn.Write([]byte(msg))
        }
		break
	case ElevatorMap:
		data := data.(ElevatorMap)
		msg := EncodeElevatorMap(data)
		if (conn != nil) {
	        conn.Write([]byte(msg))
        }
	}
}

//just print the error message
func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s", err.Error())
		os.Exit(1)
	}
}
