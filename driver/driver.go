/* This file contains high level functions for elevator interaction */

package driver

import . "math"


/*
-----------------------------
-------- Constants ----------
-----------------------------
*/

const N_FLOORS = 4
const N_BUTTONS = 3

//event types
const (
	BUTTON_CALL_UP = 0
	BUTTON_CALL_DOWN = 1
	BUTTON_COMMAND = 2
	JOB_DONE = 3
)

//pack lamp and button channels into matrices to be able to loop through them
const  lamp_channel_matrix = [N_FLOORS][N_BUTTONS]int {
	{LIGHT_DOWN1, LIGHT_UP1, LIGHT_COMMAND1},
	{LIGHT_DOWN2, LIGHT_UP2, LIGHT_COMMAND2},
	{LIGHT_DOWN3, LIGHT_UP3, LIGHT_COMMAND3},
	{LIGHT_DOWN4, LIGHT_UP4, LIGHT_COMMAND4},
}

const  button_channel_matrix = [N_FLOORS][N_BUTTONS]int {
    {FLOOR_UP1, FLOOR_DOWN1, FLOOR_COMMAND1},
    {FLOOR_UP2, FLOOR_DOWN2, FLOOR_COMMAND2},
    {FLOOR_UP3, FLOOR_DOWN3, FLOOR_COMMAND3},
    {FLOOR_UP4, FLOOR_DOWN4, FLOOR_COMMAND4},
}


/*
-----------------------------
------ Types/Structs --------
-----------------------------
*/


type button_type int

type Event struct{
    EventType int
    Floor int
}


/*
-----------------------------
-------- Functions ----------
-----------------------------
*/



func Elev_init() int{
    
    // Init hardware
    if (Io_init() == 0) {
        return 0;
	}
    // Zero all floor button lamps
    for i := 0; i < N_FLOORS; i++ {
        if (i != 0) {
			//There is no CALL_DOWN button at first floor
            Set_button_lamp(BUTTON_CALL_DOWN, i, 0);
        }

        if (i != N_FLOORS-1) {
			//There is no CALL_UP button at the top floor
            Set_button_lamp(BUTTON_CALL_UP, i, 0);
        }

       Set_button_lamp(BUTTON_COMMAND, i, 0);
    }

    // Clear stop lamp, door open lamp, and set floor indicator to ground floor.
    Set_stop_lamp(0);
    Set_door_open_lamp(0);
    Set_floor_indicator(0);

    // Return success.
    return 1;
}

func Set_floor_indicator(floor int ){

    // Binary encoding. One light must always be on.
    if (floor & 0x02 == 0x02){
        Io_set_bit(FLOOR_IND1);
    } else {
        Io_clear_bit(FLOOR_IND1);
    }
        
    if (floor & 0x01 == 0x01) {
        Io_set_bit(FLOOR_IND2);
    } else {
        Io_clear_bit(FLOOR_IND2);
    }
}


func Set_button_lamp( button button_type, floor int , value int ){ 
	if value == 1 {
	    Io_set_bit(lamp_channel_matrix[floor][button]);
    } else {
        Io_clear_bit(lamp_channel_matrix[floor][button]);    
    }
}



func Set_stop_lamp(value int){
    if (value == 1) {
        Io_set_bit(LIGHT_STOP);
    } else {
        Io_clear_bit(LIGHT_STOP);
    }
}


func Set_door_open_lamp(value int){
    if (value == 1) {
        Io_set_bit(DOOR_OPEN)
    } else {
        Io_clear_bit(DOOR_OPEN)
    }
}

//Poll all the buttons for a 
func Look_for_events() Event{
    for i:=0; i<N_FLOORS;i++{
        for j:=0; j<N_BUTTONS;j++{
            if(Io_read_bit(button_channel_matrix[i][j])==1){
                return Event{j,i}
            }
        }
    }
    return Event{-1,-1}
}


// last_speed needs to be "static"
var last_speed = 0;

func Set_speed( speed int){
    // In order to sharply stop the elevator, the direction bit is toggled
    // before setting speed to zero.
    
    // If to start (speed > 0)
    if (speed > 0){
        Io_clear_bit(MOTORDIR);
    } else if (speed < 0){
        Io_set_bit(MOTORDIR);
    }else{
		// If to stop (speed == 0)
		if (last_speed < 0){
		    Io_clear_bit(MOTORDIR);
		}else if (last_speed > 0){
		    Io_set_bit(MOTORDIR);
        }
    }

    last_speed = speed ;

    // Write new setting to motor.
    Io_write_analog(MOTOR, int(2048 + 4*Abs(float64(speed))));
}

