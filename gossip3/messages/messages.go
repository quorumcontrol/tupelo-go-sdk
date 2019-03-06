package messages

import (
	"fmt"
	"reflect"

	"github.com/tinylib/msgp/msgp"
)

// WireMessage represents a message to send over the wire.
type WireMessage interface {
	msgp.Marshaler
	msgp.Unmarshaler
	TypeCode() int8
}

var messageRegister = map[int8]reflect.Type{}

// RegisterMessage registers a message type.
func RegisterMessage(msg WireMessage) {
	typ := reflect.TypeOf(msg).Elem()
	messageRegister[msg.TypeCode()] = typ
}

// GetUnmarshaler gets an unmarshaler for the corresponding type code.
func GetUnmarshaler(typeCode int8) (WireMessage, error) {
	typ, ok := messageRegister[typeCode]
	if !ok {
		return nil, fmt.Errorf("could not find type: %d", typeCode)
	}
	msg := reflect.New(typ).Interface()
	return msg.(WireMessage), nil
}
