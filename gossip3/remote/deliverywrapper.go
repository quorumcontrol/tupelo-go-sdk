package remote

import (
	"github.com/gogo/protobuf/proto"
	wbridge "github.com/quorumcontrol/messages/build/go/bridge"
	"github.com/quorumcontrol/tupelo-go-sdk/tracing"
)

type wireDeliveryWrapper struct {
	tracing.ContextHolder
	*wbridge.WireDelivery
	originalMessage proto.Message
}
