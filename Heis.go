/*This is the main code for controlling the elevator. Each elevator is a slave to a master,
as long as there is a network connection. */

package main

import (
	. "./datatypes"
	. "./driver"
	. "./encoder"
	. "./network"
	"fmt"
	"io/ioutil"
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
	[4]int{0, 0, 0, 0},
	[4]int{0, 0, 0, 0},
	0,
	STOP,
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

//Receives events through a channel and makes appropriate actions
func handleEvents(eventChan chan Event, updateMasterChan chan bool, jobDoneChan chan Event, startChan chan int, stopChan chan bool) {
	for {

		//wait for an event
		event := <-eventChan
		if event.Floor >= 0 && event.Floor < N_FLOORS {

			switch event.EventType {

			//Received new jobs, update the correct job list
			case BUTTON_CALL_DOWN:
				elev.Downrun[event.Floor] = CALL
				Set_button_lamp(BUTTON_CALL_DOWN, event.Floor, 1)
				if elev.Dir == STOP {
					//if this is the only job:
					_, dir := findNextJob()
					fmt.Println(dir)
					startChan <- dir
					elev.Dir = dir
				}
				break
			case BUTTON_CALL_UP:
				Set_button_lamp(BUTTON_CALL_UP, event.Floor, 1)
				elev.Uprun[event.Floor] = CALL
				if elev.Dir == STOP {
					//if this is the only job:
					_, dir := findNextJob()
					startChan <- dir
					elev.Dir = dir
				}
				break
			case BUTTON_COMMAND:
				if event.Floor < elev.Current_floor {
					elev.Downrun[event.Floor] = COMMAND
				} else if event.Floor >= elev.Current_floor {
					elev.Uprun[event.Floor] = COMMAND
				}
				if elev.Dir == STOP {
					//if this is the only job:
					_, dir := findNextJob()
					startChan <- dir
					elev.Dir = dir
				}
				//write backup to file
				ioutil.WriteFile("localBU.txt", []byte(EncodeElevatorStruct(elev)), 0644)
				break

			//Events from the elevator
			case PASSED_FLOOR:
				elev.Current_floor = event.Floor
				Set_floor_indicator(event.Floor)
				//check if we should stop here
				if elev.Dir == UP && (elev.Uprun[event.Floor] > 0 || event.Floor == N_FLOORS-1) || (elev.Dir == DOWN && (elev.Downrun[event.Floor] > 0 || event.Floor == 0)) {
					stopChan <- true
					elev.Uprun[event.Floor] = 0
					elev.Downrun[event.Floor] = 0
					Set_button_lamp(BUTTON_CALL_DOWN, event.Floor, 0)
					Set_button_lamp(BUTTON_CALL_UP, event.Floor, 0)
					Set_button_lamp(BUTTON_COMMAND, event.Floor, 0)
					ioutil.WriteFile("localBU.txt", []byte(EncodeElevatorStruct(elev)), 0644)
					jobDoneChan <- Event{JOB_DONE, event.Floor}
				}
				break
			case JOB_REQUEST:
				//elevator is ready for a new job, so give a new order!
				job, dir := findNextJob()
				elev.Dir = dir
				if job {
					startChan <- dir
				}
				break

			//Light control events
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

			//send updated information to the master
			updateMasterChan <- true
			<-updateMasterChan
		}
	}
}

//Controls the movement of the elevator
func elevator(eventChan chan Event, startChan chan int, stopChan chan bool) {
	floor := -1
	eventChan <- Event{JOB_REQUEST, 0}

	for {
		//wait until there is something to do
		dir := <-startChan

		//start the elevator
		Set_door_open_lamp(0)
		Set_speed(dir * 100)

		stopped := false
		for !stopped {
			select {
			//keep moving until receiving a stop signal
			case <-stopChan:
				stopped = true
				Set_speed(STOP)
				Set_door_open_lamp(1)
				break
			default:
				//while moving, keep polling the sensors and send PASSED_FLOOR events when passing floors
				for i := 0; i < N_FLOORS; i++ {
					if Io_read_bit(Sensor[i]) == 1 && floor != i {
						floor = i
						eventChan <- Event{PASSED_FLOOR, i}
					}
				}
				Sleep(1 * Millisecond)
				break
			}
		}
		//wait a few seconds to allow people to get off
		Sleep(2 * Second)

		//ready for a new order
		eventChan <- Event{JOB_REQUEST, 0}
	}
}

//Returns the direction to the next job
func findNextJob() (bool, int) {
	up_max := -1
	up_min := -1
	down_max := -1
	down_min := -1

	for i := 0; i < N_FLOORS; i++ {
		if elev.Uprun[i] > 0 {
			up_max = i
		}
		if elev.Uprun[N_FLOORS-1-i] > 0 {
			up_min = N_FLOORS - 1 - i
		}
		if elev.Downrun[i] > 0 {
			down_max = i
		}
		if elev.Downrun[N_FLOORS-1-i] > 0 {
			down_min = N_FLOORS - 1 - i
		}
	}

	if elev.Dir == UP || elev.Dir == STOP {
		//if elevator is moving upwards, prioritise upgoing jobs above the current position
		if up_max > elev.Current_floor {
			return true, UP
		}
		//if no upgoing jobs above, its time to look at the downgoing jobs.
		//Start to see if there are downgoing jobs above the current position, and create a upgoing job to that floor
		if down_max > elev.Current_floor {
			elev.Uprun[down_max] = COMMAND
			return true, UP
		}
		//if neither upgoing nor downgoing jobs above, see if there are downgoing jobs below
		if down_max >= 0 {
			return true, DOWN
		}
		//and see if there is upgoing jobs below
		if up_min < elev.Current_floor && up_min >= 0 {
			elev.Downrun[up_min] = COMMAND
			return true, DOWN
		}
	} else if elev.Dir == DOWN {
		//if elevator is moving downwards, prioritise downgoin jobs below the current position
		if down_min < elev.Current_floor && down_min >= 0 {
			return true, DOWN
		}
		//if no more downgoing jobs below, its time to look at the upgoing jobs.
		//Start to see if there are upgoing jobs below the current position, and create a downgoing job to that floor
		if up_min < elev.Current_floor && up_min >= 0 {
			elev.Downrun[up_min] = COMMAND
			return true, DOWN
		}
		//if neither upgoing nor downgoing jobs below, see if there are upgoing jobs above
		if up_max >= 0 {
			return true, UP
		}
		//and see if there is downgoing jobs above
		if down_max > elev.Current_floor {
			elev.Uprun[down_max] = COMMAND
			return true, UP
		}
	}
	//no jobs are found
	return false, STOP
}

//polls the elevator control panel
func lookForLocalEvents(newEventChan chan Event) {
	for {
		event := Poll_buttons()
		//make sure event is of right type and not occuring at the current floor
		if event.EventType >= 0 && event.EventType < 3 && Io_read_bit(Sensor[event.Floor]) == 0 {
			newEventChan <- event
		}
		Sleep(10 * Millisecond)
	}
}

//interprets and handles messages from the master (received from network module)
func handleMasterMessage(msgChan chan Message, handleEventChan chan Event, jobDoneChan chan Event) {
	for {
		m := <-msgChan
		//first bit of the message is an id telling us what type of message it is
		msgType, _ := strconv.Atoi(string(m.Msg[0]))
		switch msgType {
		case HANDSHAKE:
			//Nothing to handle.
			break
		case QUEUENUMBER:
			//The place in the queue to become the new master if anything goes wrong
			masterQueue, _ = strconv.Atoi(m.Msg[1:])
			fmt.Println("Received place in master queue:", masterQueue)
			break
		case EVENT:
			//Event (Job order)
			e := DecodeEvent(m.Msg[1:])
			fmt.Println("Received order, type", e.EventType, "at floor", e.Floor)
			if Io_read_bit(Sensor[e.Floor]) == 0 || (e.EventType == TURN_OFF_LIGHTS) {
				handleEventChan <- e
			} else if e.EventType >= 0 && e.EventType < 2 {
				//ignore the order if the elevator already is at that floor. Signal that the job is done.
				jobDoneChan <- Event{JOB_DONE, e.Floor}
			}
			break
		case BACKUP:
			//backup of the masters elevator map
			backup = DecodeElevatorMap(m.Msg[1:])
			break
		}
	}
}

func main() {
	//network status flag
	var NetworkMode = bool(true)
	
	//Initialize the elevator and print status
	success, floor := Elev_init()
	if success != 1 {
		fmt.Println("Could not initialize elevator. Exiting...")
		return
	}
	fmt.Println("Elevator initialised!")
	elev.Current_floor = floor
	
	//Read backup file
	bs, err := ioutil.ReadFile("localBU.txt")
	if err == nil {
		fmt.Println("Reading backup: ")
		temp := DecodeElevatorStruct(string(bs[1:]))
		for i := 0; i < N_FLOORS; i++ {
			if temp.Uprun[i] == COMMAND {
				elev.Uprun[i] = COMMAND
				Set_button_lamp(BUTTON_COMMAND, i, 1)
			}
			if temp.Downrun[i] == COMMAND {
				elev.Downrun[i] = COMMAND
				Set_button_lamp(BUTTON_COMMAND, i, 1)
			}
		}
		fmt.Println(elev)
	}

	//Try to find master
	fmt.Println("Searching for master...")
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
		//if no connection can be made, something is wrong with the network. Start the program in non network mode instead
		fmt.Println("Could not connect to master. Starting program in non network mode")
		NetworkMode = false
		go HandleLostConnection(-1, msgChan, lostMasterChan, newMasterChan)
	} else {
		//send info about self to master
		SendMessage(conn, elev)
	}

	//create channels to send events between threads
	localEventChan := make(chan Event)
	handleEventChan := make(chan Event)
	updateMasterChan := make(chan bool)
	jobDoneChan := make(chan Event)
	startChan := make(chan int)
	stopChan := make(chan bool)

	//start threads that control the elevator
	go handleEvents(handleEventChan, updateMasterChan, jobDoneChan, startChan, stopChan)
	go elevator(handleEventChan, startChan, stopChan)

	//start threads that look for local events and handles messages from the master
	go lookForLocalEvents(localEventChan)
	go handleMasterMessage(msgChan, handleEventChan, jobDoneChan)

	//Wait for something to happen.
	for {
		select {
		case event := <-localEventChan:
			if !NetworkMode || event.EventType == BUTTON_COMMAND {
				//handle all events localy if not running in NetworkMode
				Set_button_lamp(event.EventType, event.Floor, 1)
				handleEventChan <- event
			} else {
				//send events to the master
				SendMessage(conn, event)
			}
			break
		case event := <-jobDoneChan:
			//notify the master that a job has been completed
			if NetworkMode && event.EventType == JOB_DONE {
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
