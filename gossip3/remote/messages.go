//go:generate msgp

package remote

import (
	"fmt"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/quorumcontrol/tupelo-go-client/gossip3/messages"
)

type ActorPID struct {
	Address string
	Id      string
}

func (ActorPID) TypeCode() int8 {
	return -10
}

func ToActorPid(a *actor.PID) *ActorPID {
	if a == nil {
		return nil
	}
	return &ActorPID{
		Address: a.Address,
		Id:      a.Id,
	}
}

func FromActorPid(a *ActorPID) *actor.PID {
	return actor.NewPID(a.Address, a.Id)
}

type DestinationSettable interface {
	SetDestination(*ActorPID)
	GetDestination() *ActorPID
}

type WireDelivery struct {
	Header            map[string]string
	Message           []byte
	Type              int8
	Target            *ActorPID
	Sender            *ActorPID
	Outgoing          bool `msg:"-"`
	SerializedContext map[string]string
}

// GetMessage deserializes a WireMessage.
func (wd *WireDelivery) GetMessage() (messages.WireMessage, error) {
	msg, err := messages.GetUnmarshaler(wd.Type)
	if err != nil {
		return nil, err
	}
	if _, err = msg.UnmarshalMsg(wd.Message); err != nil {
		return nil, fmt.Errorf("error unmarshaling message: %s", err)
	}
	return msg, nil
}
