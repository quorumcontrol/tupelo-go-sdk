//go:generate msgp

package remote

import (
	"fmt"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/quorumcontrol/tupelo-go-client/gossip3/messages"
	"github.com/quorumcontrol/tupelo-go-client/gossip3/serializer"
	"github.com/tinylib/msgp/msgp"
)

type remoteDeliver struct {
	header       actor.ReadonlyMessageHeader
	message      interface{}
	target       *actor.PID
	sender       *actor.PID
	serializerID int32
}

func toWireDelivery(rd *remoteDeliver) *wireDelivery {
	marshaled, err := rd.message.(msgp.Marshaler).MarshalMsg(nil)
	if err != nil {
		panic(fmt.Errorf("could not marshal message: %v", err))
	}
	wd := &wireDelivery{
		Message: marshaled,
		Type:    serializer.GetTypeCode(rd.message),
		Target:  messages.ToActorPid(rd.target),
		Sender:  messages.ToActorPid(rd.sender),
	}
	if rd.header != nil {
		wd.Header = rd.header.ToMap()
	}
	return wd
}

type registerBridge struct {
	Peer    string
	Handler *actor.PID
}

type wireDelivery struct {
	Header   map[string]string
	Message  []byte
	Type     int8
	Target   *messages.ActorPID
	Sender   *messages.ActorPID
	Outgoing bool `msg:"-"`
}

func (wd *wireDelivery) GetMessage() (msgp.Unmarshaler, error) {
	msg := serializer.GetUnmarshaler(wd.Type)
	_, err := msg.UnmarshalMsg(wd.Message)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling message: %v", err)
	}
	return msg, nil
}
