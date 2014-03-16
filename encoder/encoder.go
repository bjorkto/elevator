package encoder

//This package contains function for encoding/decoding the messages that are sent over the network

import (
	. "../datatypes"
	"strconv"
	"strings"
)

func EncodeEvent(e Event) string {
	msg := "2" + strconv.Itoa(e.EventType) + " " + strconv.Itoa(e.Floor) + "\n"
	return msg
}

func DecodeEvent(msg string) Event {
	var e Event
	fields := strings.Fields(msg)
	e.EventType, _ = strconv.Atoi(fields[0])
	e.Floor, _ = strconv.Atoi(fields[1])
	return e
}

func EncodeElevatorStruct(elev ElevatorStruct) string {
	msg := "3"
	for i := 0; i < N_FLOORS; i++ {
		msg += strconv.Itoa(elev.Uprun[i]) + " "
	}
	for i := 0; i < N_FLOORS; i++ {
		msg += strconv.Itoa(elev.Downrun[i]) + " "
	}
	msg += strconv.Itoa(elev.Current_floor) + " "
	msg += strconv.Itoa(elev.Dir) + "\n"
	return msg
}

func DecodeElevatorStruct(msg string) ElevatorStruct {
	var estruct ElevatorStruct
	fields := strings.Fields(msg)
	for i := 0; i < N_FLOORS; i++ {
		estruct.Uprun[i], _ = strconv.Atoi(fields[i])
	}
	for i := 0; i < N_FLOORS; i++ {
		estruct.Downrun[i], _ = strconv.Atoi(fields[i+N_FLOORS])
	}
	estruct.Current_floor, _ = strconv.Atoi(fields[2*N_FLOORS])
	estruct.Dir, _ = strconv.Atoi(fields[2*N_FLOORS+1])
	estruct.Conn = nil
	return estruct
}

func EncodeElevatorMap(emap ElevatorMap) string {
	msg := "4"
	for addr, elev := range(emap){
	    msg += addr + " "
	    for i := 0; i < N_FLOORS; i++ {
		    msg += strconv.Itoa(elev.Uprun[i]) + " "
	    }
	    for i := 0; i < N_FLOORS; i++ {
		    msg += strconv.Itoa(elev.Downrun[i]) + " "
	    }
	    msg += strconv.Itoa(elev.Current_floor) + " "
	    msg += strconv.Itoa(elev.Dir) + " "
	 }
	 msg += "\n"
	 return msg
}

func DecodeElevatorMap(msg string) ElevatorMap {
	var emap = make(ElevatorMap)
	fields := strings.Fields(msg)
	fields_per_elevator := (3+2*N_FLOORS)
	number_of_elevators := len(fields)/fields_per_elevator
	
	for n := 0; n < number_of_elevators; n++ {
	    addr := fields[n*fields_per_elevator] 
	    var estruct ElevatorStruct
	    for i := 0; i < N_FLOORS; i++ {
		    estruct.Uprun[i], _ = strconv.Atoi(fields[n*fields_per_elevator + 1 + i])
	    }
	    for i := 0; i < N_FLOORS; i++ {
		    estruct.Downrun[i], _ = strconv.Atoi(fields[n*fields_per_elevator + 1 + i + N_FLOORS])
	    }
	    estruct.Current_floor, _ = strconv.Atoi(fields[n*fields_per_elevator + 1 + 2*N_FLOORS])
	    estruct.Dir, _ = strconv.Atoi(fields[n*fields_per_elevator + 1 + 2*N_FLOORS + 1])
	    estruct.Conn = nil
        emap[addr] = estruct
    }          	    
	return emap
}
