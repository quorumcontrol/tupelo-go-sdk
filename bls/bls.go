package bls

import (
	"fmt"
	"log"

	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/kyber/v3/pairing/bn256"
	dedisbls "go.dedis.ch/kyber/v3/sign/bls"
	"go.dedis.ch/kyber/v3/util/random"
)

var suite = bn256.NewSuite()

type SignKey struct {
	private kyber.Scalar
	verKey  *VerKey
	value   []byte
}

type VerKey struct {
	value  []byte
	public kyber.Point
}

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

func MustNewSignKey() *SignKey {
	key, err := NewSignKey()
	if err != nil {
		panic(fmt.Sprintf("error generating key: %v", err))
	}
	return key
}

func (sk *SignKey) Bytes() []byte {
	return sk.value
}

func (sk *SignKey) Sign(msg []byte) ([]byte, error) {
	return dedisbls.Sign(suite, sk.private, msg)
}

func (sk *SignKey) VerKey() (*VerKey, error) {
	return sk.verKey, nil
}

func (sk *SignKey) MustVerKey() *VerKey {
	verKey, err := sk.VerKey()
	if err != nil {
		log.Panicf("error getting verKey: %v", err)
	}
	return verKey
}

func (vk *VerKey) Bytes() []byte {
	return vk.value
}

func (vk *VerKey) Verify(sig, msg []byte) (bool, error) {
	err := dedisbls.Verify(suite, vk.public, msg, sig)
	if err != nil {
		return false, nil
	}
	return true, nil
}

func SumSignatures(sigs [][]byte) ([]byte, error) {
	return dedisbls.AggregateSignatures(suite, sigs...)
}

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
