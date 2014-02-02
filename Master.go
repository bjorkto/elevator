package main

import (
        "fmt"
        "net"
        "os"
		"time"
)

/*
------------------------
-------Constants--------
------------------------
*/

//event types
const (
	CALL_ELEVATOR_UP = 0		//someone calls the elevator, up direction
	CALL_ELEVATOR_DOWN = 1		//someone calls the elevator, down direction	
	FLOOR_REQUEST = 2			//A button on the "choose floor" panel have been pushed
	SENSOR = 3					//Elevator passes a sensor
	JOB_DONE = 4				//Elevator has completed a job
)


/*
------------------------
-------Structures-------
------------------------
*/

//struct containing information about an event
type event struct {
	eventType int8	//what type of event?
	floor int8		//where did the event take place?
}

//struct containing the status of an elevator
type elevator struct{
	jobList []int8
	currentFloor int8
}


/*
------------------------
----Global Variables----
------------------------
*/

//Hashtable/map of currently connected elevators
var emap = make(map[net.Addr]elevator)


/*
------------------------
-------Functions--------
------------------------
*/


//Error handling
func checkError(err error){
        if err != nil {
                fmt.Fprintf(os.Stderr,"Error: %s", err.Error())
                os.Exit(1)
        }
}


//Broadcast on UDP to signal that a master exists
func broadcastUDP(service string){
    
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
func startTCPServer(port string){
	    
	tcpAddr, err := net.ResolveTCPAddr("tcp4", port)
    checkError(err)
    listener, err := net.ListenTCP("tcp", tcpAddr)
    checkError(err)
	
	fmt.Println("TCP server started, listening for connections...")

	//listen for inncoming connections and spawn new threads to handle each client
	for{
		conn, err := listener.Accept()		//accept connection
		if err != nil {
				continue
		}
		fmt.Println("Accepted connection from: ", conn.RemoteAddr())	//print address of the client
			
		//create entry in elevator map
		emap[conn.RemoteAddr()] = elevator{jobList: []int8{}, currentFloor: 0}
			
		//spawn go-routine that reads messages from the client 
		go ListenToClient(conn)
	}
}


//Listen for messages from the client
func ListenToClient(conn net.Conn){
	addr := conn.RemoteAddr()
	for{
		var buf [2]byte
        _, err := conn.Read(buf[0:])		//read message
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		e := event{eventType: int8(buf[0]), floor: int8(buf[1])}	//convert from []byte to event struct
        fmt.Print(addr, " says: ") 
		switch(e.eventType){
			case CALL_ELEVATOR_UP:
				fmt.Println("Request for upgoing elevator in floor", e.floor)
				break
			case CALL_ELEVATOR_DOWN:
				fmt.Println("Request for downgoing elevator in floor", e.floor)
				break
			case FLOOR_REQUEST:
				fmt.Println("Request for moving elevator to floor", e.floor)
				break
			case SENSOR:
				fmt.Println("Elevator just passed floor", e.floor)
				break
			case JOB_DONE:
				fmt.Println("Completed job at floor", e.floor)
				break
			default:
				fmt.Println("Unknown event...")
				break
		}
    }
}


//Main function
func main(){
	go broadcastUDP("192.168.0.255:10001")
	startTCPServer(":10002")
}
