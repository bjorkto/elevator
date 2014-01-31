package main

import (
        "fmt"
        "net"
        "os"
		"time"
)

//Structs
type elevator struct{
	jobList []int8
	currentFloor int8
}

//Global variabels
var emap = make(map[net.Addr]elevator)

//Error handling
func checkError(err error){
        if err != nil {
                fmt.Fprintf(os.Stderr,"Error: %s", err.Error())
                os.Exit(1)
        }
}

func broadcast(service string){
    
	//Broadcast on UDP to signal that a master exists
	fmt.Println("Broadcastin on UDP ", service)
    udpAddr, err := net.ResolveUDPAddr("udp4", service)
    checkError(err)
    conn, err := net.DialUDP("udp", nil, udpAddr)
    checkError(err)
		
	for{
		conn.Write([]byte("Stian og Vegards heiser"))
		time.Sleep(500*time.Millisecond)
    }
	
}


func main(){
	broadcast("192.168.0.255:10001")
}
