package main

import (
        "fmt"
		. "time"
		. "./network"
		. "./driver"
)

/*
-----------------------------
-------- Constants ----------
-----------------------------
*/


//pack the sensor channels into a matrix so it is easy to loop through them
var sensor =[N_FLOORS]int {SENSOR1, SENSOR2, SENSOR3, SENSOR4}


/*
-----------------------------
------ Types/Structs --------
-----------------------------
*/

//contains info about the status of the elevator
type elevatorStruct struct{
	uprun [N_FLOORS]bool
	downrun [N_FLOORS]bool
	current_floor int
	dir int
}


/*
-----------------------------
--------- Globals -----------
-----------------------------
*/

//create an elevatorStruct for the elevator connected to this program
//(only one thread will ever have write access)

var elev = elevatorStruct{
	[4]bool{false, false, false, false},
	[4]bool{false, false, false, false},
	0,
	0,
}


/*
-----------------------------
-------- Functions ----------
-----------------------------
*/



//Receives events through a channel and adds/removes jobs 

func handleJobArrays(eventChan chan Event){
    for{
	
		//wait for an event
        event := <- eventChan
		
		//update the correct array depending on event type
	    if (event.EventType==BUTTON_CALL_DOWN) {
            elev.downrun[event.Floor]=true
        }else if(event.EventType==BUTTON_CALL_UP) {
            elev.uprun[event.Floor]=true
        }else if (event.EventType == BUTTON_COMMAND) {
            if(event.Floor<elev.current_floor){
                elev.downrun[event.Floor]=true
            }else if (event.Floor>elev.current_floor){
                elev.uprun[event.Floor]=true
            }
        }else if(event.EventType == JOB_DONE){
            elev.uprun[event.Floor] = false
            elev.downrun[event.Floor] = false
            Set_button_lamp(BUTTON_CALL_DOWN, event.Floor, 0)
            Set_button_lamp(BUTTON_CALL_UP, event.Floor, 0)
            Set_button_lamp(BUTTON_COMMAND, event.Floor, 0)
        } 
        fmt.Println(elev.uprun, '\n', elev.downrun)
    }  
}


//Scans the joblist between lower_floor and upper_floor and returns true if there is a job
func isJobs(joblist [4]bool, lower_floor int, upper_floor int) (bool, int) {
	for i := lower_floor ; i < upper_floor; i++{
            if (joblist[i]){
				return true, i
        }            
    }
	return false, -1
}


//Controls the elevator, sends a JOB_DONE event when a job is completed
func elevator(eventChan chan Event){

	
	for{
        //need to sleep a bit because the go runtime is stupid...
        Sleep(1*Millisecond)
				
		//Is it upgoing jobs above the current floor?
        up, _ := isJobs(elev.uprun, elev.current_floor+1, N_FLOORS)
        
		//Is it upgoing jobs below the current floor?
        if up_below, i := isJobs(elev.uprun, 0, elev.current_floor); up_below == true{
			
			//create a downgoing event...?
			eventChan <- Event{BUTTON_CALL_DOWN, i}
		}
		
		//Is it downgoing jobs below the current floor?
        down, _ := isJobs(elev.downrun, 0, elev.current_floor)
        
		//Is it downgoing jobs above the current floor?
        if down_above, i := isJobs(elev.downrun, elev.current_floor+1, N_FLOORS); down_above == true{
			
			//create an upgoing job... ?
            eventChan <- Event{BUTTON_CALL_UP, i}
        }             
        
		//While going up:       
        for up{
			
			//full speed ahead
			elev.dir = 1
            Set_speed(100)
				
			//Keep polling the sensors until job is completed
            complete := false
            for !complete {
                for i:=elev.current_floor+1;i<N_FLOORS;i++{
                	Sleep(1*Millisecond)
                    if (Io_read_bit(sensor[i]) == 1) {
						//update current floor when passing a sensor
						elev.current_floor=i
						Set_floor_indicator(i)
						if (i == N_FLOORS-1 || elev.uprun[i]){
							//stop if there is a job or we are at the top floor
							complete = true
							break
                        }
                    }
                }
			}
			
			//stop!!!
            Set_speed(0)
			elev.dir=0
				
			//signal that the job is complete
            eventChan <- Event{JOB_DONE, elev.current_floor}
            Sleep(2*Second)
                
            //Is there still more upgoing jobs above?
            up, _ = isJobs(elev.uprun, elev.current_floor+1, N_FLOORS)				              
        }
        
        //If going down
		for down{
           	//Full speed ahead!
			elev.dir = -1;
            Set_speed(-100)
				
			//Keep polling the sensors until job is completed
			complete := false
            for !complete {
                for i:=0;i<elev.current_floor;i++{
                	Sleep(1*Millisecond)
                    if (Io_read_bit(sensor[i]) == 1) {
						//update current floor when passing a sensor
                        elev.current_floor=i
                        Set_floor_indicator(i)
						if (i == 0 || elev.downrun[i]){
							//stop if there is a job or we are at the bottom floor
							complete = true
							break
						}
					}
                }
            }
				
			//stop!!!!
            Set_speed(0)
            elev.dir=0
				
			//signal that the job is done
			eventChan <- Event{JOB_DONE, elev.current_floor}
            Sleep(2*Second)
                
            //Are there still more downgoing jobs below?
			down, _ = isJobs(elev.downrun, 0, elev.current_floor-1)
			
        }
    //Let the cycle begin again
	}
}




func main(){
    //Try to find master
	exists, address := SearchForMaster(":10001")
	
	//spawn a new master process if not found
	if (!exists){
		StartNewMaster()
		exists, address = SearchForMaster(":10001")
	}else{
		fmt.Println("Master found at", address)
	}
	
	
	//Connect to Master (TCP)
	conn := ConnectToMaster(address)
	if (conn == nil){
		fmt.Println("Could not connect to master. Exiting...")
		return
	}
	
	
	//Init elevator and print status
	success := Elev_init()
	if success != 1 {
		fmt.Println ("Could not initialize elevator. Exiting..." )
		return
	}
	

	//create channel to send events
	eventChan := make(chan Event)
	
	//start threads to handle the jobs and the elevator
	go handleJobArrays(eventChan)
	go elevator(eventChan)
	
	//Debugging only: Set networkmode to false to handle all events locally
	NetworkMode = false
	
	//Wait for someone to push a button
	for{
	    event := Look_for_events()
	    //make sure event is of right type and not occuring at the current floor
	    if (event.EventType >= 0 && event.EventType < 3 && Io_read_bit(sensor[event.Floor]) == 0){
	        Set_button_lamp(event.EventType, event.Floor, 1)
			if !NetwokMode{
				//handle all events localy if not running in NetworkMode
				eventChan <- event
			} else {
			
				//send events to the master
			}
	    }
        Sleep(10*Millisecond)
    }	
}
