package bls

import (
	"bytes"
	"fmt"
	"log"

	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/kyber/v3/pairing/bn256"
	dedisbls "go.dedis.ch/kyber/v3/sign/bls"
	"go.dedis.ch/kyber/v3/util/random"
)

var suite = bn256.NewSuite()

// SignKey represents a signkey.
type SignKey struct {
	private kyber.Scalar
	verKey  *VerKey
	value   []byte
}

// VerKey represents a verkey.
type VerKey struct {
	value  []byte
	public kyber.Point
}

// BytesToSignKey converts a byte array to a SignKey.
func BytesToSignKey(keyBytes []byte) *SignKey {
	scalar := suite.G2().Scalar()
	err := scalar.UnmarshalBinary(keyBytes)
	if err != nil {
		panic(fmt.Sprintf("invalid sign key: %v", err))
	}
	public := suite.G2().Point().Mul(scalar, nil)
	verKeyBytes, _ := public.MarshalBinary()
	return &SignKey{
		private: scalar,
		value:   keyBytes,
		verKey: &VerKey{
			value:  verKeyBytes,
			public: public,
		},
	}
}

// BytesToVerKey converts a byte array to a VerKey.
func BytesToVerKey(keyBytes []byte) *VerKey {
	point := suite.G2().Point()
	err := point.UnmarshalBinary(keyBytes)
	if err != nil {
		panic(fmt.Sprintf("invalid verkey bytes: %v", err))
	}
	return &VerKey{
		public: point,
		value:  keyBytes,
	}
}

// NewSignKey instantiates a new SignKey.
func NewSignKey() (*SignKey, error) {
	private, public := dedisbls.NewKeyPair(suite, random.New())
	privBytes, err := private.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("error marshaling: %v", err)
	}
	pubBytes, err := public.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("error marshaling: %v", err)
	}
	return &SignKey{
		private: private,
		value:   privBytes,
		verKey: &VerKey{
			public: public,
			value:  pubBytes,
		},
	}, nil
}

// MustNewSignKey is like NewSignKey, but will panic on error.
func MustNewSignKey() *SignKey {
	key, err := NewSignKey()
	if err != nil {
		panic(fmt.Sprintf("error generating key: %v", err))
	}
	return key
}

// Bytes converts a SignKey to a byte array.
func (sk *SignKey) Bytes() []byte {
	return sk.value
}

// Sign signs a message.
func (sk *SignKey) Sign(msg []byte) ([]byte, error) {
	return dedisbls.Sign(suite, sk.private, msg)
}

// VerKey gets the VerKey for a SignKey.
func (sk *SignKey) VerKey() (*VerKey, error) {
	return sk.verKey, nil
}

// MustVerKey is like VerKey except it panics on error.
func (sk *SignKey) MustVerKey() *VerKey {
	verKey, err := sk.VerKey()
	if err != nil {
		log.Panicf("error getting verKey: %v", err)
	}
	return verKey
}

// Bytes gets the bytes for a VerKey.
func (vk *VerKey) Bytes() []byte {
	return vk.value
}

// Verify verifies a message given a signature.
func (vk *VerKey) Verify(sig, msg []byte) (bool, error) {
	err := dedisbls.Verify(suite, vk.public, msg, sig)
	if err != nil {
		return false, nil
	}
	return true, nil
}

// SumSignatures aggregates signatures.
func SumSignatures(sigs [][]byte) ([]byte, error) {
	firstSig := sigs[0]
	var aggregateSigs [][]byte
	var endSigs [][]byte
	for i, sig := range sigs {
		if i > 0 && bytes.Equal(firstSig, sig) {
			endSigs = append(endSigs, sig)
			continue
		}
		aggregateSigs = append(aggregateSigs, sig)
	}
	aggregateSigs = append(aggregateSigs, endSigs...)
	return dedisbls.AggregateSignatures(suite, aggregateSigs...)
}

// There's a weird bug where if the first key is repeated: https://github.com/dedis/kyber/issues/400
// that's why we have to do this weird "append to back" thing
func SumVerKeys(verKeys []*VerKey) (*VerKey, error) {
	first := verKeys[0]
	var appendToBack []kyber.Point

	var points []kyber.Point
	for i, verKey := range verKeys {
		if i > 0 && verKey.public.Equal(first.public) {
			fmt.Println("equal")
			appendToBack = append(appendToBack, verKey.public)
			continue
		}
		points = append(points, verKey.public)
	}
	points = append(points, appendToBack...)
	aggregatedPublic := dedisbls.AggregatePublicKeys(suite, points...)
	pubBytes, err := aggregatedPublic.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("error marshaling: %v", err)
	}
	return &VerKey{
		public: aggregatedPublic,
		value:  pubBytes,
	}, nil
}

// VerifyMultiSig verifies a message using a multi signature.
//TODO: let's pass in real verkeys and not binary
func VerifyMultiSig(sig, msg []byte, verKeys [][]byte) (bool, error) {
	points := make([]kyber.Point, len(verKeys))
	for i, verKeyBytes := range verKeys {
		p := suite.G2().Point()
		err := p.UnmarshalBinary(verKeyBytes)
		if err != nil {
			return false, fmt.Errorf("error unmarshaling: %v", err)
		}
		points[i] = p
	}
	aggregatedPublic := dedisbls.AggregatePublicKeys(suite, points...)
	err := dedisbls.Verify(suite, aggregatedPublic, msg, sig)
	if err != nil {
		return false, nil
	}
	return true, nil
}
