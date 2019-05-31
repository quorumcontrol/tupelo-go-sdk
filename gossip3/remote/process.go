package remote

import (
	"fmt"
	"log"
	"reflect"
	mbridge "github.com/quorumcontrol/messages/build/go/bridge"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/quorumcontrol/tupelo-go-sdk/tracing"
	"github.com/golang/protobuf/proto"
	ptypes "github.com/golang/protobuf/ptypes"

)

type process struct {
	pid     *actor.PID
	gateway *actor.PID
}

func newProcess(pid, gateway *actor.PID) actor.Process {
	return &process{
		pid:     pid,
		gateway: gateway,
	}
}

// Send user message, implements actor.Process.
func (ref *process) SendUserMessage(pid *actor.PID, message interface{}) {
	header, msg, sender := actor.UnwrapEnvelope(message)
	wireMsg, ok := msg.(proto.Message)
	if !ok {
		log.Printf("error sending user message, message doesn't implement proto.Message: %s\n",
			reflect.TypeOf(msg))
		return
	}
	sendMessage(ref.gateway, pid, header, wireMsg, sender, -1)
}

func sendMessage(gateway, pid *actor.PID, header actor.ReadonlyMessageHeader, message proto.Message, sender *actor.PID, serializerID int32) {
	marshaled, err := ptypes.MarshalAny(message)
	if err != nil {
		panic(fmt.Errorf("could not marshal message: %v", err))
	}

	wd := &mbridge.WireDelivery{
		// originalMessage: message,
		Message:         marshaled,
		// Type:            message.TypeCode(),
		Target:          toActorPid(pid),
		Sender:          toActorPid(sender),
	}
	if header != nil {
		wd.Header = header.ToMap()
	}
	wd.Outgoing = true

	if tracing.Enabled {
		traceable, ok := message.(tracing.Traceable)
		if ok && traceable.Started() {
			traceable.NewSpan("process-sendMessage").Finish()
		}
	}

	actor.EmptyRootContext.Send(gateway, wd)
}

func (ref *process) SendSystemMessage(pid *actor.PID, message interface{}) {
	//intercept any Watch messages and direct them to the endpoint manager
	switch msg := message.(type) {
	case *actor.Watch:
		panic("remote watching unsupported")
		// rw := &remoteWatch{
		// 	Watcher: msg.Watcher,
		// 	Watchee: pid,
		// }
		// endpointManager.remoteWatch(rw)
	case *actor.Unwatch:
		panic("remote unwatching unsupported")
		// ruw := &remoteUnwatch{
		// 	Watcher: msg.Watcher,
		// 	Watchee: pid,
		// }
		// endpointManager.remoteUnwatch(ruw)
	case proto.Message:
		sendMessage(ref.gateway, pid, nil, msg, nil, -1)
	default:
		log.Printf("error sending system message, not convertible to WireMessage: %s\n",
			reflect.TypeOf(message))
		return
	}
}

func (ref *process) Stop(pid *actor.PID) {
	panic("remote stop is unsupported")
}


func toActorPid(a *actor.PID) *mbridge.ActorPID {
	if a == nil {
		return nil
	}
	return &mbridge.ActorPID{
		Address: a.Address,
		Id:      a.Id,
	}
}

func fromActorPid(a *mbridge.ActorPID) *actor.PID {
	return actor.NewPID(a.Address, a.Id)
}