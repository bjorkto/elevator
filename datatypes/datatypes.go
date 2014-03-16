package datatypes

import "net"

//This file containts types and constants that are used in several source files

const N_FLOORS = 4
const N_BUTTONS = 3

//event types
const (
	BUTTON_CALL_UP   = 0
	BUTTON_CALL_DOWN = 1
	BUTTON_COMMAND   = 2
	JOB_DONE         = 3
	PASSED_FLOOR     = 4
	DIRECTION_CHANGE_UP = 5
	DIRECTION_CHANGE_DOWN = 6
	DIRECTION_CHANGE_STOP = 7
	TURN_ON_UP_LIGHT = 8
	TURN_ON_DOWN_LIGHT = 9
	TURN_OFF_LIGHTS = 10
)

//button types
const (
	CALL	= 1
	COMMAND = 2
)

type ElevatorStruct struct {
	Uprun         [N_FLOORS]int
	Downrun       [N_FLOORS]int
	Current_floor int
	Dir           int
	Conn          *net.TCPConn
}

type ElevatorMap map[string]ElevatorStruct

type Event struct {
	EventType int
	Floor     int
}

type Message struct {
	Sender *net.TCPConn
	Msg    string
}
