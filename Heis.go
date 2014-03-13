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
)

/*
-----------------------------
--------- Globals -----------
-----------------------------
*/

//create an elevatorStruct for the elevator connected to this program
//(only one thread will ever have write access)

var elev = ElevatorStruct{
	[4]bool{false, false, false, false},
	[4]bool{false, false, false, false},
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

func handleJobArrays(eventChan chan Event, updateMasterChan chan bool) {
	for {

		//wait for an event
		event := <-eventChan

		//update the correct array depending on event type
		if event.EventType == BUTTON_CALL_DOWN {
			elev.Downrun[event.Floor] = true
		} else if event.EventType == BUTTON_CALL_UP {
			elev.Uprun[event.Floor] = true
		} else if event.EventType == BUTTON_COMMAND {
			if event.Floor < elev.Current_floor {
				elev.Downrun[event.Floor] = true
			} else if event.Floor > elev.Current_floor {
				elev.Uprun[event.Floor] = true
			}
		} else if event.EventType == JOB_DONE {
			elev.Uprun[event.Floor] = false
			elev.Downrun[event.Floor] = false
			Set_button_lamp(BUTTON_CALL_DOWN, event.Floor, 0)
			Set_button_lamp(BUTTON_CALL_UP, event.Floor, 0)
			Set_button_lamp(BUTTON_COMMAND, event.Floor, 0)
		} else if event.EventType == PASSED_FLOOR {
			elev.Current_floor = event.Floor
			Set_floor_indicator(event.Floor)
		} else if event.EventType == DIRECTION_CHANGE {
			elev.Dir = event.Floor //Not really a floor. It's actually the direction
		}

		updateMasterChan <- true
	}
}

//Scans the joblist between lower_floor and upper_floor and returns true if there is a job
func isJobs(joblist [4]bool, lower_floor int, upper_floor int) (bool, int) {
	for i := lower_floor; i < upper_floor; i++ {
		if joblist[i] {
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
			eventChan <- Event{DIRECTION_CHANGE, 0}
		}

		//While going up:
		for up {

			if elev.Dir != 1 {
				eventChan <- Event{DIRECTION_CHANGE, 1}
			}

			//full speed ahead
			Set_speed(100)

			//Keep polling the sensors until job is completed
			complete := false
			for !complete {
				for i := elev.Current_floor + 1; i < N_FLOORS; i++ {
					Sleep(1 * Millisecond)
					if Io_read_bit(Sensor[i]) == 1 {
						//update current floor when passing a sensor
						eventChan <- Event{PASSED_FLOOR, i}
						if i == N_FLOORS-1 || elev.Uprun[i] {
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
			Sleep(2 * Second)

			//Is there still more upgoing jobs above?
			up, _ = isJobs(elev.Uprun, elev.Current_floor+1, N_FLOORS)
		}

		//While going down
		for down {

			if elev.Dir != -1 {
				eventChan <- Event{DIRECTION_CHANGE, -1}
			}

			//Full speed ahead!
			Set_speed(-100)

			//Keep polling the sensors until job is completed
			complete := false
			for !complete {
				for i := 0; i < elev.Current_floor; i++ {
					Sleep(1 * Millisecond)
					if Io_read_bit(Sensor[i]) == 1 {
						//update current floor when passing a sensor
						eventChan <- Event{PASSED_FLOOR, i}
						if i == 0 || elev.Downrun[i] {
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
			Sleep(2 * Second)

			//Are there still more downgoing jobs below?
			down, _ = isJobs(elev.Downrun, 0, elev.Current_floor-1)

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
			if !(event.Floor == elev.Current_floor && event.EventType == BUTTON_COMMAND) {
				Set_button_lamp(event.EventType, event.Floor, 1)
				newEventChan <- event
			}
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
			handleEventChan <- DecodeEvent(m.Msg[1:])
			break
		case 4:
			//backup of the masters elevator map
			backup = DecodeElevatorMap(m.Msg[1:])
			break
		}
	}
}

func main() {

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
	}

	//Initialize the elevator and print status
	success, floor := Elev_init()
	if success != 1 {
		fmt.Println("Could not initialize elevator. Exiting...")
		return
	}

	//send info about self to master
	elev.Current_floor = floor
	SendMessage(conn, elev)

	//create channels to send events between threads
	localEventChan := make(chan Event)
	handleEventChan := make(chan Event)
	updateMasterChan := make(chan bool)

	//start threads that control the elevator
	go handleJobArrays(handleEventChan, updateMasterChan)
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
		case <-updateMasterChan:
			//send updated info about the elevator to the master
			if NetworkMode {
				SendMessage(conn, elev)
			}
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
