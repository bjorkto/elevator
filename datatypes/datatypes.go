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
	DIRECTION_CHANGE = 5
)

type ElevatorStruct struct {
	Uprun         [N_FLOORS]bool
	Downrun       [N_FLOORS]bool
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
