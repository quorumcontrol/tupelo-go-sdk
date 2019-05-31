//go:generate msgp

package remote

// import (
// 	"fmt"

// 	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/messages"
// )

// type WireDelivery struct {
// 	originalMessage messages.WireMessage

// 	Header   map[string]string
// 	Message  []byte
// 	Type     int8
// 	Target   *messages.ActorPID
// 	Sender   *messages.ActorPID
// 	Outgoing bool `msg:"-"`
// 	Tracing  map[string]string
// }

// // GetMessage deserializes a WireMessage.
// func (wd *WireDelivery) GetMessage() (messages.WireMessage, error) {
// 	msg, err := messages.GetUnmarshaler(wd.Type)
// 	if err != nil {
// 		return nil, err
// 	}
// 	if _, err = msg.UnmarshalMsg(wd.Message); err != nil {
// 		return nil, fmt.Errorf("error unmarshaling message: %s", err)
// 	}
// 	return msg, nil
// }
