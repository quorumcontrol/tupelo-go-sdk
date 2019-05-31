package consensus

import (
	"github.com/quorumcontrol/messages/build/go/services"
	"fmt"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/golang/protobuf/proto"
)

func RequestID(req *services.AddBlockRequest) []byte {
	//TODO: fix me and make me canonical
	bits,err := proto.Marshal(req)
	if err != nil {
				panic(fmt.Errorf("error marshaling: %v", err))
			}
			return  crypto.Keccak256(bits)
}