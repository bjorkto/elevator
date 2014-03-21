/*This file containts types and constants that are used in several source files*/

package datatypes

import "net"

const N_FLOORS = 4
const N_BUTTONS = 3

//event types
const (
	BUTTON_CALL_UP        = 0
	BUTTON_CALL_DOWN      = 1
	BUTTON_COMMAND        = 2
	JOB_DONE              = 3
	PASSED_FLOOR          = 4
	TURN_ON_UP_LIGHT      = 5
	TURN_ON_DOWN_LIGHT    = 6
	TURN_OFF_LIGHTS       = 7
	JOB_REQUEST           = 8
)

//button types
const (
	CALL    = 1
	COMMAND = 2
)

//Direction types
const (
	UP   = 1
	DOWN = -1
	STOP = 0
)

//Network message types
const (
	HANDSHAKE   = 0
	QUEUENUMBER = 1
	EVENT = 2
	ELEV_INFO = 3
	BACKUP = 4
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
