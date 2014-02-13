package main

import (
        "fmt"
		"os/exec"
		"time"
		. "./network"
		. "./driver"
)

/*
-----------------------------
-------- Constants ----------
-----------------------------
*/


//pack the sensor channels into a matrix so it is easy to loop through them
const sensor =[N_FLOOR]int {SENSOR1, SENSOR2, SENSOR3, SENSOR4}


/*
-----------------------------
------ Types/Structs --------
-----------------------------
*/

//contains info about the status of the elevator
type elevatorStruct struct{
	uprun [N_FLOOR]bool
	downrun [N_FLOOR]bool
	current_floor int
	dir int
}


/*
-----------------------------
--------- Globals -----------
-----------------------------
*/

//create a elevatorStruct for the elevator connected to this program
//(only one thread will ever have write access)

var elev = elevatorStruct{
	{false, false, false, false},
	{false, false, false, false},
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
            if(event.Floor<current_floor){
                elev.downrun[event.Floor]=true
            }else if (event.Floor>current_floor){
                elev.uprun[event.Floor]=true
            }
        }else if(event.EventType == JOB_DONE){
            elev.uprun[event.Floor] = false
            elev.downrun[event.Floor] = false
        } 
    }  
}


//Scans the joblist between lower_floor and upper_floor and returns true if there is a job
func isJobs(joblist []bool, int lower_floor, int upper_floor) bool {
	for i := lower_floor ; i < upper_floor; i++{
            if (joblist[i]){
				return true
        }            
    }
	return false
}


//Controls the elevator, sends a JOB_DONE event when a job is completed
func elevator(eventChan chan Event){

    var up = bool(false)
	var down = bool(false)
	
	for{
        //need to sleep a bit because the go runtime is stupid...
        Sleep(1*Millisecond)
				
		//Is it upgoing jobs above the current floor?
        up = isJobs(elev.uprun, elev.current_floor+1, N_FLOOR)
        
		//Is it upgoing jobs below the current floor?
        if isJobs(elev.uprun, 0, elev.current_floor-1){
			
			//create a downgoing event...?
			eventChan <- Event{BUTTON_CALL_DOWN, i}
		}
		
		//Is it downgoing jobs below the current floor?
        down = isJobs(elev.downrun, 0, elev.current_floor-1)
        
		//Is it downgoing jobs above the current floor?
        if isJobs(elev.downrun, elev.current_floor+1, N_FLOOR){
			
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
                for i:=elev.current_floor+1;i<N_FLOOR;i++{
                    if (Io_read_bit(sensor[i]) == 1) {
						//update current floor when passing a sensor
						elev.current_floor=i
						if (i == N_FLOOR-1 || elev.uprun[i]){
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
            up = isJobs(elev.uprun, elev.current_floor+1, N_FLOOR)				              
        }
        
        //If going down
		for down{
           	//Full speed ahead!
			dir = -1;
            Set_speed(-100)
				
			//Keep polling the sensors until job is completed
			complete := false
            for !complete {
                for i:=0;i<elev.current_floor;i++{
                    if (Io_read_bit(sensor[i]) == 1) {
						//update current floor when passing a sensor
                        elev.current_floor=i
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
			down = isJobs(elev.downrun, 0, elev.current_floor-1)
			
        }
    //Let the cycle begin again
	}
}


func main(){
    
    //Try to find master
	exists, address := SearchForMaster(":10001")
	
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
	conn := ConnectToMaster(address)
	if (conn == nil){
		fmt.Println("Could not connect to master...")
		return
	}
	
	
	//Init elevator and print status
	fmt.Println(Elev_init())
	
	//create channel to send events
	eventChan := make(chan Event)
	
	//start threads to handle the jobs and the elevator
	go handleJobArrays(eventChan)
	go elevator(eventChan)
	
	//Wait for someone to push a button
	for{
	    event := Look_for_events()
	    if (event.EventType >= 0 && event.EventType <= 3){
	        eventChan <- event
	    }
        Sleep(10*Millisecond)
    }
	
}
