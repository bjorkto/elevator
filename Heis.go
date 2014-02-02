package main

import (
        "fmt"
        "net"
        "os"
		"time"
		"os/exec"
		"strings"
		"math/rand"
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


//Check if master exists by listening for UDP message
func searchForMaster() (bool, string) {
	
    service := ":10001"
    fmt.Println("Listening for master...")
    udpAddr, err := net.ResolveUDPAddr("udp4", service)
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

func main(){
    
    //Try to find master
	exists, address := searchForMaster()
	
	//spawn a new master process if not found
	if (!exists){
		fmt.Println("Master not found! Starting new master...")
		cmd := exec.Command("mate-terminal", "-x", "./Master")
		err := cmd.Start()
		fmt.Println(err.Error())
		address = "localhost:10002"
		time.Sleep(1*time.Second)  //Give the new process some time to set up the server
	}else{
		fmt.Println("Master found at", address)
	}
	

	//Connect to Master (TCP)
	service := address + ":10002"
	fmt.Println("Attemting to connect to master...", )
	tcpAddr, err := net.ResolveTCPAddr("tcp4", service)
	checkError(err)
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	checkError(err)
	fmt.Println("Connection established! \nSending events...")
	
	//Send random events every two seconds
	for{
		time.Sleep(2*time.Second)
		e := event{eventType: int8(rand.Intn(5)), floor: int8(rand.Intn(4) + 1)}	//create a random event struct
		bytearray := []byte{byte(e.eventType), byte(e.floor)}
		conn.Write(bytearray)
	}
	
	var dummy string
	fmt.Scanln(&dummy)
}
