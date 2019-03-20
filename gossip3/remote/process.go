package remote

import (
	"fmt"
	"log"
	"reflect"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/quorumcontrol/tupelo-go-client/gossip3/messages"
	"github.com/quorumcontrol/tupelo-go-client/gossip3/middleware"
	"github.com/quorumcontrol/tupelo-go-client/tracing"
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
	wireMsg, ok := msg.(messages.WireMessage)
	if !ok {
		log.Printf("error sending user message, message doesn't implement messages.WireMessage: %s\n",
			reflect.TypeOf(msg))
		return
	}
	sendMessage(ref.gateway, pid, header, wireMsg, sender, -1)
}

func sendMessage(gateway, pid *actor.PID, header actor.ReadonlyMessageHeader, message messages.WireMessage, sender *actor.PID, serializerID int32) {
	var serializedContext map[string]string

	if tracing.Enabled {
		traceableMsg, ok := message.(tracing.Traceable)

		if ok {
			traceableMsg.NewSpan("sendMessage").Finish()

			serialized, err := traceableMsg.SerializedContext()
			if err == nil {
				serializedContext = serialized
			} else {
				middleware.Log.Errorw("error serializing", "err", err)
			}
		}
	}

	marshaled, err := message.MarshalMsg(nil)
	if err != nil {
		panic(fmt.Errorf("could not marshal message: %v", err))
	}
	wd := &WireDelivery{
		Message:           marshaled,
		Type:              message.TypeCode(),
		Target:            ToActorPid(pid),
		Sender:            ToActorPid(sender),
		SerializedContext: serializedContext,
	}
	if header != nil {
		wd.Header = header.ToMap()
	}
	wd.Outgoing = true

	gateway.Tell(wd)
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
	case messages.WireMessage:
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
