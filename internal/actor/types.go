package actor

import (
	"github.com/asynkron/protoactor-go/actor"
)

var managerPID *actor.PID

// SetManagerPID sets the global ManagerActor PID
func SetManagerPID(pid *actor.PID) {
	managerPID = pid
}

// GetManagerPID gets the global ManagerActor PID
func GetManagerPID() *actor.PID {
	return managerPID
}
