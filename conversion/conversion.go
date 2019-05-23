package conversion

import (
	"fmt"

	"github.com/Workiva/go-datastructures/bitarray"
	"github.com/quorumcontrol/messages/build/go/signatures"
	extmsgs "github.com/quorumcontrol/tupelo-go-sdk/gossip3/messages"
)

func ToExternalSignature(s *signatures.Signature) (*extmsgs.Signature, error) {
	signers := bitarray.NewBitArray(uint64(len(s.Signers)))
	for i, didSign := range s.Signers {
		if didSign {
			if err := signers.SetBit(uint64(i)); err != nil {
				return nil, fmt.Errorf("error setting bit: %v", err)
			}
		}
	}
	marshalledSigners, err := bitarray.Marshal(signers)
	if err != nil {
		return nil, fmt.Errorf("error marshalling signers array: %v", err)
	}

	return &extmsgs.Signature{
		ObjectID:    s.ObjectId,
		PreviousTip: s.PreviousTip,
		NewTip:      s.NewTip,
		View:        s.View,
		Cycle:       s.Cycle,
		Type:        s.Type,
		Signers:     marshalledSigners,
		Signature:   s.Signature,
		Height:      s.Height,
	}, nil
}

func ToInternalSignature(sig extmsgs.Signature) (*signatures.Signature, error) {
	var signersArray []bool
	if sig.Signers != nil {
		signers, err := bitarray.Unmarshal(sig.Signers)
		if err != nil {
			return nil, fmt.Errorf("error unmarshalling signers array: %v", err)
		}

		signersArray := make([]bool, signers.Capacity())
		for i := uint64(0); i < signers.Capacity(); i++ {
			isSet, err := signers.GetBit(i)
			if err != nil {
				return nil, fmt.Errorf("error getting signer from bitarray: %v", err)
			}

			signersArray[i] = isSet
		}
	}

	return &signatures.Signature{
		ObjectId:    sig.ObjectID,
		PreviousTip: sig.PreviousTip,
		NewTip:      sig.NewTip,
		View:        sig.View,
		Cycle:       sig.Cycle,
		Signers:     signersArray,
		Signature:   sig.Signature,
		Type:        sig.Type,
		Height:      sig.Height,
	}, nil
}
