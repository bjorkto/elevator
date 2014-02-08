package main

import (
		. "./driver"
		. "fmt"
		. "time"
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

//Controls the elevator, sends a JOB_DONE event when a job is completed
func elevator(eventChan chan Event){
    for{
        //need to sleep a bit because the go runtime is stupid...
        Sleep(1*Millisecond)
		
		//Is it upgoing jobs above the current floor?
        for i:=elev.current_floor+1;i<N_FLOOR;i++{
            if (elev.uprun[i]){
				//go up
                elev.dir=1
            }            
        }
        
		//Is it upgoing jobs belove the current floor?
        for i:=0; i<elev.current_floor; i++{
            if (elev.uprun[i]){
				//create a downgoing job... ?
                eventChan <- Event{BUTTON_CALL_DOWN, i}
            }            
        }        
        
		//If going up:
        if dir==1{
            
            for{
				//full speed ahead
                Set_speed(100)
				
				//when passing a sensor, check if we should stop
                complete := false
                for !complete {
                    for i:=elev.current_floor+1;i<N_FLOOR;i++{
                        if (Io_read_bit(sensor[i]) == 1) && elev.uprun[i]{
                            elev.current_floor=i
                            complete = true
                            break
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
                for i:=elev.current_floor;i<N_FLOOR;i++{
                    if (elev.uprun[i]){
						//still going up
                        elev.dir=1
                    } 
                }
				
				//If there are no more upgoing jobs, stop the loop
                if elev.dir==0{
                    break
                }
                
            }
        }
        
        
        //Is it downgoing jobs below the current floor?
        for i:=0;i<elev.current_floor;i++{
            if (elev.downrun[i]){
				//going down
                elev.dir=-1
            }            
        }
        
		//Is it downgoing jobs above the current floor?
        for i:=elev.current_floor+1;i<4;i++{
            if (elev.downrun[i]){
				//create an upgoing job... ?
                eventChan <- Event{BUTTON_CALL_UP, i}
            }            
        } 
        
        //If going down
		if elev.dir==-1{
            for{
				//Full speed ahead!
                Set_speed(-100)
				
				//when passing a sensor, check if we should stop
                complete := false
                for !complete {
                    for i:=0;i<elev.current_floor;i++{
                        if (Io_read_bit(sensor[i]) == 1) && elev.downrun[i]{
                            elev.current_floor=i
                            complete = true
                            break
                        }
                    }
                }
				
				//stop!!!!
                Set_speed(0)
                elev.dir=0
				
				//signal that the job is done
				eventChan <- Event{JOB_DONE, elev.current_floor}
                Sleep(2*Second)
                
                //Are there still more downgoing jobs?
                for i:=0;i<elev.current_floor;i++{
                    if (elev.downrun[i]){
						//still going down
                        elev.dir=-1
                    } 
                }
				
				//If no more jobs, stop the loop
                if elev.dir==0{
                    break
                }
                
            }
        }
    //Let the cycle begin again
	}
}

func main(){

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


