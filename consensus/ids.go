package consensus

import (
	"github.com/quorumcontrol/messages/build/go/signatures"
	"encoding/binary"
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

func ConflictSetID(objectID []byte, height uint64) string {
	return string(crypto.Keccak256(append(objectID, uint64ToBytes(height)...)))
}

func GetSignable(sig signatures.Signature) []byte {
	return append(append(sig.ObjectId, append(sig.PreviousTip, sig.NewTip...)...), append(uint64ToBytes(sig.View), uint64ToBytes(sig.Cycle)...)...)
}


func uint64ToBytes(id uint64) []byte {
	a := make([]byte, 8)
	binary.BigEndian.PutUint64(a, id)
	return a
}