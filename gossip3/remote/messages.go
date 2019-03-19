//go:generate msgp

package remote

import (
	"fmt"

	"github.com/quorumcontrol/tupelo-go-client/gossip3/messages"
)

type WireDelivery struct {
	Header            map[string]string
	Message           []byte
	Type              int8
	Target            *messages.ActorPID
	Sender            *messages.ActorPID
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
