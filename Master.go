package main

import (
		. "./network"
		"net"
		"fmt"
		"strconv"
		. "math"
		. "./datatypes"
		"strings"
)


var emap = make(ElevatorMap)

/*
-----------------------------
--------- Functions ---------
-----------------------------
*/

func handleMessages(msgChan chan Message){

   for {
      m:= <- msgChan
      //first bit is an id telling us what type of message it is
      msgType, _ := strconv.Atoi(string(m.Msg[0]))
      switch msgType{
			case 0:
				//type 0 is only a handshake. Nothing to handle. 
				break
			case 2:
				//Event
				fields := strings.Fields(m.Msg[1:])
				var e Event
				e.EventType, _ = strconv.Atoi(fields[0])
				e.Floor, _ = strconv.Atoi(fields[1])
				fmt.Println("Event at", m.Sender.RemoteAddr())
				fmt.Println("Type:", e.EventType, "Floor:", e.Floor)
			      
				if e.EventType == BUTTON_COMMAND{
				   //Send directly back to sender
				   SendMessage(m.Sender, e)
				}else {
				   //Find most suitable elevator to handle the event
			      mostSuitable := findMostSuitable(e, emap)
			      SendMessage(mostSuitable, e)
			   }
				break
			case 3:
			   //update on information from elevator
			   var estruct ElevatorStruct
			   fields := strings.Fields(m.Msg[1:])
			   for i:= 0; i< N_FLOORS; i++{
			      estruct.Uprun[i], _ = strconv.ParseBool(fields[i])
			   }
			   for i:= 0; i< N_FLOORS; i++{
			      estruct.Downrun[i], _ = strconv.ParseBool(fields[i+N_FLOORS])
			   }
			   estruct.Current_floor, _ = strconv.Atoi(fields[2*N_FLOORS])
			   estruct.Dir, _ = strconv.Atoi(fields[2*N_FLOORS+1])
			   emap[m.Sender] = estruct
			   fmt.Println(emap[m.Sender])
		}     
   }
}


func findMostSuitable(buttonEvent Event, emap ElevatorMap) (*net.TCPConn){

	floor := buttonEvent.Floor+1
	dir:=0
	if buttonEvent.EventType==0{
		dir = 1
	}else if buttonEvent.EventType==1{
		dir = -1
	}else {
		return nil
	}

	bestDist := 99999
	var bestElev *net.TCPConn = nil
	tempDist := 0
	maxFloor := -1

	for key, elevator := range emap {	
		if (dir==elevator.Dir || elevator.Dir==0){
			tempDist = int(Abs(float64(elevator.Current_floor-floor)))
			if tempDist<bestDist{
				bestDist=tempDist
				bestElev=key;
			}else if tempDist==bestDist && elevator.Dir==0{
				bestDist=tempDist
				bestElev=key;
			}
		}else{
			if elevator.Dir==1{
				for j:=len(elevator.Uprun)-1; j>=0;j-- {
					if (elevator.Uprun[j] == true){
						maxFloor=j+1
						break
					}
				}
			}else{
				for j:=0; j<len(elevator.Downrun);j++ {
					if (elevator.Downrun[j] == true){
						maxFloor=j+1
						break
					}
				}
			}
			tempDist = int(Abs(float64(elevator.Current_floor-floor))+2*Abs(float64(maxFloor-elevator.Current_floor)))
			if tempDist<bestDist{
				bestDist=tempDist
				bestElev=key;
		   }
	   }
	}
	return bestElev;
}



func main(){
	//create a elevatorMap to keep track of all connected clients
	
	//channel to receive information about new connections
	newconnChan := make(chan *net.TCPConn)
	delconnChan := make(chan *net.TCPConn)
	msgChan := make(chan Message)
	//start the TCP server in its own thread
	go StartTCPServer(":10002", newconnChan, delconnChan, msgChan)
	
	//start the UDP broadcasting in its own thread
	go BroadcastUDP("129.241.187.255:10001")
	
	go handleMessages(msgChan)
	
	//for each new connection, make a new entry in the elevatorMap
	for{
		select{
			case newConn := <- newconnChan:
				emap[newConn] = ElevatorStruct{[4]bool{false, false, false, false}, [4]bool{false, false, false, false}, 0, 0}
				fmt.Println("Number of connections: ", len(emap))
				idMsg := "1" + strconv.Itoa(len(emap));
				newConn.Write([]byte(idMsg)) 
				break
			case delConn := <- delconnChan:
				delete(emap, delConn)
				fmt.Println("Number of connections: ", len(emap))
				break
		}
	}
}
