package wrapper

import (
	"context"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip4/client2/pubsubinterfaces"
)

type libp2pWrapper struct {
	*pubsub.PubSub
}

func (lpw *libp2pWrapper) Subscribe(topic string) (pubsubinterfaces.Subscription, error) {
	sub, err := lpw.PubSub.Subscribe(topic)
	return &libp2pSubscriptionWrapper{sub}, err
}

// wraps a libp2p pubsub instance into the needed client interfaces
func WrapLibp2p(libp2ppubsub *pubsub.PubSub) pubsubinterfaces.Pubsubber {
	return &libp2pWrapper{
		PubSub: libp2ppubsub,
	}
}

type libp2pSubscriptionWrapper struct {
	*pubsub.Subscription
}

func (lpsw *libp2pSubscriptionWrapper) Next(ctx context.Context) (pubsubinterfaces.Message, error) {
	return lpsw.Next(ctx)
}
