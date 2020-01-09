package remote

import (
	"github.com/gogo/protobuf/proto"
	wbridge "github.com/quorumcontrol/messages/v2/build/go/bridge"
	"github.com/quorumcontrol/tupelo-go-sdk/tracing"
)

type wireDeliveryWrapper struct {
	tracing.ContextHolder
	*wbridge.WireDelivery
	originalMessage proto.Message
}
