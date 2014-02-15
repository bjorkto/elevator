package main

import (
		. "./network"
		"net"
		"fmt"
		"strconv"
)

/*
------------------------------
------- Structs/Types --------
------------------------------
*/

type elevatorStruct struct{
	uprun [4]bool
	downrun [4]bool
	current_floor int
	dir int
}

type elevatorMap map[*net.TCPConn]elevatorStruct


/*
-----------------------------
--------- Functions ---------
-----------------------------
*/



//TODO
//func findMostSuitable() - Loop through the elevatorMap to find the most suitable elevator for the job


func main(){
	//create a elevatorMap to keep track of all connected clients
	emap := make(elevatorMap)
	
	//channel to receive information about new connections
	newconnChan := make(chan *net.TCPConn)
	delconnChan := make(chan *net.TCPConn)
	//start the TCP server in its own thread
	go StartTCPServer(":10002", newconnChan, delconnChan)
	
	//start the UDP broadcasting in its own thread
	go BroadcastUDP("129.241.187.255:10001")
	
	//for each new connection, make a new entry in the elevatorMap
	for{
		select{
			case newConn := <- newconnChan:
				emap[newConn] = elevatorStruct{[4]bool{false, false, false, false}, [4]bool{false, false, false, false}, 0, 0}
				fmt.Println("Number of connections: ", len(emap))
				newConn.Write([]byte(strconv.Itoa(len(emap)))) //MÅ UTVIDES TIL Å SENDE ID FOR MLDINGSTYP
				break
			case delConn := <- delconnChan:
				delete(emap, delConn)
				fmt.Println("Number of connections: ", len(emap))
				break
		}
	}
}
