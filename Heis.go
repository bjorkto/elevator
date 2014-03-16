package main

import (
	. "./datatypes"
	. "./driver"
	. "./encoder"
	. "./network"
	"fmt"
	"net"
	"strconv"
	. "time"
	 "io/ioutil"
)

/*
-----------------------------
--------- Globals -----------
-----------------------------
*/

//create an elevatorStruct for the elevator connected to this program
//(only one thread will ever have write access)

var elev = ElevatorStruct{
	[4]int{0,0,0,0},
	[4]int{0,0,0,0},
	0,
	0,
	nil,
}

var masterQueue = int(1)

//Backup of all elevators connected to master
var backup = make(ElevatorMap)

/*
-----------------------------
-------- Functions ----------
-----------------------------
*/

//Receives events through a channel and updates the elevator structure accordingly

func handleJobArrays(eventChan chan Event, updateMasterChan chan bool, jobDoneChan chan Event) {
	for {

		//wait for an event
		event := <-eventChan
		if (event.Floor >= 0 && event.Floor < N_FLOORS) {
			//update the correct array depending on event type
			switch event.EventType {
				case (BUTTON_CALL_DOWN):
					elev.Downrun[event.Floor] = CALL
					break
				case (BUTTON_CALL_UP):
					elev.Uprun[event.Floor] = CALL
					break
				case (BUTTON_COMMAND):
					if event.Floor < elev.Current_floor {
						elev.Downrun[event.Floor] = COMMAND
					} else if event.Floor >= elev.Current_floor {
						elev.Uprun[event.Floor] = COMMAND
					}
					//lagre til fil
					ioutil.WriteFile("localBU.txt", []byte(EncodeElevatorStruct(elev)), 0644) 
					break
				case JOB_DONE:
					elev.Uprun[event.Floor] = 0
					elev.Downrun[event.Floor] = 0
					Set_button_lamp(BUTTON_CALL_DOWN, event.Floor, 0)
					Set_button_lamp(BUTTON_CALL_UP, event.Floor, 0)
					Set_button_lamp(BUTTON_COMMAND, event.Floor, 0)
					ioutil.WriteFile("localBU.txt", []byte(EncodeElevatorStruct(elev)), 0644) 
					jobDoneChan <- event
					break
				case PASSED_FLOOR:
					elev.Current_floor = event.Floor
					Set_floor_indicator(event.Floor)
					break
				case DIRECTION_CHANGE_UP:
					elev.Dir = 1
					break
				case DIRECTION_CHANGE_DOWN:
					elev.Dir = -1
					break
				case DIRECTION_CHANGE_STOP:
					elev.Dir = 0
					break
				case TURN_ON_UP_LIGHT:
					Set_button_lamp(BUTTON_CALL_UP, event.Floor, 1)
					break
				case TURN_ON_DOWN_LIGHT:
					Set_button_lamp(BUTTON_CALL_DOWN, event.Floor, 1)
					break
				case TURN_OFF_LIGHTS:
					Set_button_lamp(BUTTON_CALL_DOWN, event.Floor, 0)
					Set_button_lamp(BUTTON_CALL_UP, event.Floor, 0)
					break
			}
			updateMasterChan <- true
			<- updateMasterChan
		}
	}
}

//Scans the joblist between lower_floor and upper_floor and returns true if there is a job
func isJobs(joblist [4]int, lower_floor int, upper_floor int) (bool, int) {
	for i := lower_floor; i < upper_floor; i++ {
		if joblist[i] > 0 {
			return true, i
		}
	}
	return false, -1
}

//Controls the elevator, sends a JOB_DONE event when a job is completed
func elevator(eventChan chan Event) {

	for {
		//need to sleep a bit because the go runtime is stupid...
		Sleep(1 * Millisecond)

		//Is it upgoing jobs above the current floor?
		up, _ := isJobs(elev.Uprun, elev.Current_floor+1, N_FLOORS)

		//Is it upgoing jobs below the current floor?
		if up_below, i := isJobs(elev.Uprun, 0, elev.Current_floor); up_below == true {

			//create a downgoing event
			eventChan <- Event{BUTTON_CALL_DOWN, i}
		}

		//Is it downgoing jobs below the current floor?
		down, _ := isJobs(elev.Downrun, 0, elev.Current_floor)

		//Is it downgoing jobs above the current floor?
		if down_above, i := isJobs(elev.Downrun, elev.Current_floor+1, N_FLOORS); down_above == true {

			//create an upgoing job
			eventChan <- Event{BUTTON_CALL_UP, i}
		}

		if !up && !down && elev.Dir != 0 {
			eventChan <- Event{DIRECTION_CHANGE_STOP, 0}
			Set_door_open_lamp(1)
		}

		//While going up:
		for up {

			if elev.Dir != 1 {
				eventChan <- Event{DIRECTION_CHANGE_UP, 0}
				Set_door_open_lamp(0)
			}

			//full speed ahead
			Set_speed(100)

			//Keep polling the sensors until job is completed
			complete := false
			for !complete {
				for i := elev.Current_floor; i < N_FLOORS; i++ {
					Sleep(1 * Millisecond)
					if Io_read_bit(Sensor[i]) == 1{
						//update current floor when passing a sensor
						if i != elev.Current_floor {
							eventChan <- Event{PASSED_FLOOR, i}
						}
						if i == N_FLOORS-1 || elev.Uprun[i] > 0 {
							//stop if there is a job or we are at the top floor
							complete = true
							break
						}
					}
				}
			}

			//stop!!!
			Set_speed(0)
			

			//signal that the job is complete
			eventChan <- Event{JOB_DONE, elev.Current_floor}
			Set_door_open_lamp(1)
			Sleep(2 * Second)
			Set_door_open_lamp(0)

			//Is there still more upgoing jobs above?
			up, _ = isJobs(elev.Uprun, elev.Current_floor+1, N_FLOORS)
		}

		//While going down
		for down {

			if elev.Dir != -1 {
				eventChan <- Event{DIRECTION_CHANGE_DOWN, 0}
				Set_door_open_lamp(0)
			}

			//Full speed ahead!
			Set_speed(-100)

			//Keep polling the sensors until job is completed
			complete := false
			for !complete {
				for i := 0; i <= elev.Current_floor; i++ {
					Sleep(1 * Millisecond)
					if Io_read_bit(Sensor[i]) == 1 {
						//update current floor when passing a sensor
						if i != elev.Current_floor {
							eventChan <- Event{PASSED_FLOOR, i}
						}
						if i == 0 || elev.Downrun[i] > 0 {
							//stop if there is a job or we are at the bottom floor
							complete = true
							break
						}
					}
				}
			}

			//stop!!!!
			Set_speed(0)

			//signal that the job is done
			eventChan <- Event{JOB_DONE, elev.Current_floor}
			
			Set_door_open_lamp(1)
			Sleep(2 * Second)
			Set_door_open_lamp(0)

			//Are there still more downgoing jobs below?
			down, _ = isJobs(elev.Downrun, 0, elev.Current_floor)

		}
		//Let the cycle begin again
	}
}

//polls the elevator control panel
func lookForLocalEvents(newEventChan chan Event) {
	for {
		event := Poll_buttons()
		//make sure event is of right type and not occuring at the current floor
		if event.EventType >= 0 && event.EventType < 3 && Io_read_bit(Sensor[event.Floor]) == 0 {
			if (event.EventType == BUTTON_COMMAND) {
				Set_button_lamp(event.EventType, event.Floor, 1)
			}
			newEventChan <- event
		}
		Sleep(10 * Millisecond)
	}
}

//interprets and handles messages from the master (received from network module)
func handleMasterMessage(msgChan chan Message, handleEventChan chan Event) {
	for {
		m := <-msgChan
		//first bit of the message is an id telling us what type of message it is
		msgType, _ := strconv.Atoi(string(m.Msg[0]))
		switch msgType {
		case 0:
			//type 0 is only a handshake. Nothing to handle.
			break
		case 1:
			//type 1 is the place in the queue to become the new master if anything goes wrong
			masterQueue, _ = strconv.Atoi(m.Msg[1:])
			fmt.Println("Received place in master queue:", masterQueue)
			break
		case 2:
			//Event (Job order)
			e := DecodeEvent(m.Msg[1:])
			fmt.Println("Received order, type", e.EventType, "at floor", e.Floor)
			if (Io_read_bit(e.Floor) == 0){
				handleEventChan <- e
			}
			break
		case 4:
			//backup of the masters elevator map
			backup = DecodeElevatorMap(m.Msg[1:])
			break
		}
	}
}

func main() {

	//Read backup file
	bs, err := ioutil.ReadFile("localBU.txt")
	if err == nil{
		fmt.Println("Reading backup: ")
		temp := DecodeElevatorStruct(string(bs[1:]))
		for i := 0; i < N_FLOORS; i++ {
			if temp.Uprun[i] == COMMAND {
				elev.Uprun[i] = COMMAND
			}
			if temp.Downrun[i] == COMMAND {
				elev.Downrun[i] = COMMAND
			}		
		}
		fmt.Println(elev)
	}
	
	//Initialize the elevator and print status
	success, floor := Elev_init()
	if success != 1 {
		fmt.Println("Could not initialize elevator. Exiting...")
		return
	}
	elev.Current_floor = floor
	
	//Try to find master
	exists, address := SearchForMaster(":10001")

	//spawn a new master process if not found
	if !exists {
		StartNewMaster()
		exists, address = SearchForMaster(":10001")
	} else {
		fmt.Println("Master found at", address)
	}

	//create channels to communicate with network module
	msgChan := make(chan Message)
	lostMasterChan := make(chan *net.TCPConn)
	newMasterChan := make(chan *net.TCPConn)

	//Connect to Master (TCP)
	conn := ConnectToMaster(address, msgChan, lostMasterChan)
	if conn == nil {
		//if no connection can be made, somehting is wrong with the network. Start the program in non network mode instead
		fmt.Println("Could not connect to master. Starting program in non network mode")
		NetworkMode = false
		go HandleLostConnection(-1, msgChan, lostMasterChan, newMasterChan)
	}else{
		//send info about self to master
		SendMessage(conn, elev)
	}
	
	//create channels to send events between threads
	localEventChan := make(chan Event)
	handleEventChan := make(chan Event)
	updateMasterChan := make(chan bool)
	jobDoneChan := make(chan Event)

	//start threads that control the elevator
	go handleJobArrays(handleEventChan, updateMasterChan, jobDoneChan)
	go elevator(handleEventChan)

	//start threads that look for local events and handles messages from the master
	go lookForLocalEvents(localEventChan)
	go handleMasterMessage(msgChan, handleEventChan)

	//Wait for something to happen.
	for {
		select {
		case event := <-localEventChan:

			if !NetworkMode || event.EventType == BUTTON_COMMAND {
				//handle all events localy if not running in NetworkMode
				handleEventChan <- event
			} else {
				//send events to the master
				SendMessage(conn, event)
			}
			break
		case event := <- jobDoneChan:
			if NetworkMode && event.EventType == JOB_DONE{
				SendMessage(conn, event)
			}
			break
		case <-updateMasterChan:
			//send updated info about the elevator to the master
			if NetworkMode {
				SendMessage(conn, elev)
			}
			updateMasterChan <- true
			break
		case <-lostMasterChan:
			//lost connection with master. Try to resolve
			NetworkMode = false
			fmt.Println("Lost connection with Master, changing to non network mode. Trying to resolve... ")
			go HandleLostConnection(masterQueue, msgChan, lostMasterChan, newMasterChan)
			break
		case conn = <-newMasterChan:
			//got a new connection! Huzzah!
			NetworkMode = true
			fmt.Println("Returning to network mode")
			//send backup of the previous masters elevator map to new master
			SendMessage(conn, backup)
			//send info about self to new master
			SendMessage(conn, elev)
		}
	}
}
