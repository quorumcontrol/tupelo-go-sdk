package consensus

import (
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/messages/build/go/signatures"
	"github.com/quorumcontrol/messages/build/go/transactions"
	"github.com/quorumcontrol/tupelo-go-sdk/bls"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsBlockSignedBy(t *testing.T) {
	key, err := crypto.GenerateKey()
	assert.Nil(t, err)

	txn, err := chaintree.NewSetDataTransaction("down/in/the/thing", "hi")
	assert.Nil(t, err)

	blockWithHeaders := chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  nil,
			Height:       0,
			Transactions: []*transactions.Transaction{txn},
		},
	}

	signed, err := SignBlock(&blockWithHeaders, key)

	assert.Nil(t, err)

	isSigned, err := IsBlockSignedBy(signed, crypto.PubkeyToAddress(key.PublicKey).String())

	assert.Nil(t, err)
	assert.True(t, isSigned)

}

func TestVerify(t *testing.T) {
	type testCase struct {
		Description   string
		PublicKey     signatures.PublicKey
		Payload       []byte
		Signature     signatures.Signature
		ShouldSucceed bool
		ShouldError   bool
	}
	for _, testCreator := range []func() testCase{
		func() testCase {
			key, err := crypto.GenerateKey()
			assert.Nil(t, err)

			return testCase{
				Description:   "blank payload and signature",
				PublicKey:     EcdsaToPublicKey(&key.PublicKey),
				ShouldSucceed: false,
				ShouldError:   true,
			}
		},

		func() testCase {
			key, err := crypto.GenerateKey()
			assert.Nil(t, err)

			payload := []byte("hi")

			sig, err := EcdsaSign(payload, key)
			assert.Nil(t, err)

			return testCase{
				Description:   "good ecdsa signature",
				PublicKey:     EcdsaToPublicKey(&key.PublicKey),
				Payload:       payload,
				Signature:     *sig,
				ShouldSucceed: true,
				ShouldError:   false,
			}
		},

		func() testCase {
			key, err := crypto.GenerateKey()
			assert.Nil(t, err)

			payload := []byte("hi")

			sig, err := EcdsaSign([]byte("different payload"), key)
			assert.Nil(t, err)

			return testCase{
				Description:   "bad ecdsa signature",
				PublicKey:     EcdsaToPublicKey(&key.PublicKey),
				Payload:       payload,
				Signature:     *sig,
				ShouldSucceed: false,
				ShouldError:   false,
			}
		},

		func() testCase {
			key, err := bls.NewSignKey()
			assert.Nil(t, err)

			payload := crypto.Keccak256([]byte("hi"))

			sig, err := BlsSign(payload, key)
			assert.Nil(t, err)

			return testCase{
				Description:   "good bls signature",
				PublicKey:     BlsKeyToPublicKey(key.MustVerKey()),
				Payload:       payload,
				Signature:     *sig,
				ShouldSucceed: true,
				ShouldError:   false,
			}
		},

		func() testCase {
			key, err := bls.NewSignKey()
			assert.Nil(t, err)

			payload := crypto.Keccak256([]byte("hi"))

			sig, err := BlsSign([]byte("different payload"), key)
			assert.Nil(t, err)

			return testCase{
				Description:   "bad bls signature",
				PublicKey:     BlsKeyToPublicKey(key.MustVerKey()),
				Payload:       payload,
				Signature:     *sig,
				ShouldSucceed: false,
				ShouldError:   false,
			}
		},
	} {
		test := testCreator()

		hsh, err := ObjToHash(test.Payload)
		assert.Nil(t, err)

		isVerified, err := Verify(hsh, test.Signature, test.PublicKey)
		if test.ShouldError {
			assert.NotNil(t, err)
		} else {
			assert.Nil(t, err)
		}

		if test.ShouldSucceed {
			assert.True(t, isVerified, test.Description)
		} else {
			assert.False(t, isVerified, test.Description)
		}
	}
}

func TestPassPhraseKey(t *testing.T) {
	key, err := PassPhraseKey([]byte("secretPassword"), []byte("salt"))
	require.Nil(t, err)
	assert.Equal(t, []byte{0x5b, 0x94, 0x85, 0x8e, 0xda, 0x63, 0xd4, 0xe8, 0x12, 0xa5, 0xed, 0x98, 0xa2, 0xe0, 0xc8, 0xe0, 0xef, 0xcb, 0xf2, 0x72, 0x69, 0xca, 0xa2, 0x9d, 0xe9, 0x6c, 0x7a, 0x93, 0xcc, 0x73, 0x9, 0x14}, crypto.FromECDSA(key))
}
