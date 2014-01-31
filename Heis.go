package main

import (
        "fmt"
        "net"
        "os"
		"time"
		"os/exec"
		"strings"
)

//Error handling
func checkError(err error){
    if err != nil {
        fmt.Fprintf(os.Stderr,"Error: %s", err.Error())
        os.Exit(1)
    }
}

func searchForMaster() (bool, string) {
	
	//Listen for a UDP message from the master
    service := ":10001"
    fmt.Println("Listening for master...", service)
    udpAddr, err := net.ResolveUDPAddr("udp4", service)
    checkError(err)
       
    listener, err := net.ListenUDP("udp", udpAddr)
    checkError(err)

	//timeout after 5 seconds
	listener.SetReadDeadline(time.Now().Add(5*time.Second))

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
		fmt.Println("I'm my own master!")
		cmd := exec.Command("mate-terminal", "-x", "./Master")
		err := cmd.Start()
		fmt.Println(err.Error())
		address = "localhost:10002"
	}else{
		fmt.Println("Master found at", address)
	}
	
	//TODO: connect to master (TCP)
	
	var dummy string
	fmt.Scanln(&dummy)
}
