package consensus

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"fmt"
	"strings"

	"golang.org/x/crypto/pbkdf2"

	"golang.org/x/crypto/scrypt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	cbornode "github.com/ipfs/go-ipld-cbor"
	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/chaintree/dag"
	"github.com/quorumcontrol/chaintree/nodestore"
	"github.com/quorumcontrol/chaintree/safewrap"
	"github.com/quorumcontrol/chaintree/typecaster"
	"github.com/quorumcontrol/messages/build/go/signatures"
	"github.com/quorumcontrol/tupelo-go-sdk/bls"
)

func init() {
	typecaster.AddType(StandardHeaders{})
	cbornode.RegisterCborType(StandardHeaders{})
}

type SignatureMap map[string]*signatures.Signature

// Merge returns a new SignatatureMap composed of the original with the other merged in
// other wins when both SignatureMaps have signatures
func (sm SignatureMap) Merge(other SignatureMap) SignatureMap {
	newSm := make(SignatureMap)
	for k, v := range sm {
		newSm[k] = v
	}
	for k, v := range other {
		newSm[k] = v
	}
	return newSm
}

// Subtract returns a new SignatureMap with only the new signatures
// in other.
func (sm SignatureMap) Subtract(other SignatureMap) SignatureMap {
	newSm := make(SignatureMap)
	for k, v := range sm {
		_, ok := other[k]
		if !ok {
			newSm[k] = v
		}
	}
	return newSm
}

// Only returns a new SignatureMap with only keys in the keys
// slice argument
func (sm SignatureMap) Only(keys []string) SignatureMap {
	newSm := make(SignatureMap)
	for _, key := range keys {
		v, ok := sm[key]
		if ok {
			newSm[key] = v
		}
	}
	return newSm
}

const (
	KeyTypeBLSGroupSig = "BLS"
	KeyTypeSecp256k1   = "secp256k1"
)

type StandardHeaders struct {
	Signatures SignatureMap `refmt:"signatures,omitempty" json:"signatures,omitempty" cbor:"signatures,omitempty"`
}

func AddrToDid(addr string) string {
	return fmt.Sprintf("did:tupelo:%s", addr)
}

func EcdsaPubkeyToDid(key ecdsa.PublicKey) string {
	keyAddr := crypto.PubkeyToAddress(key).String()

	return AddrToDid(keyAddr)
}

func DidToAddr(did string) string {
	segs := strings.Split(did, ":")
	return segs[len(segs)-1]
}

// PublicKeyToEcdsaPub returns the ecdsa typed key from the bytes in the
// PublicKey at this time there is no error checking.
func PublicKeyToEcdsaPub(pk *signatures.PublicKey) *ecdsa.PublicKey {
	ecdsaPk, err := crypto.UnmarshalPubkey(pk.PublicKey)
	if err != nil {
		panic("Failed to unmarshal public key")
	}
	return ecdsaPk
}

// PublicKeyToAddr converts a public key to an address.
func PublicKeyToAddr(key *signatures.PublicKey) string {
	switch key.Type {
	case KeyTypeSecp256k1:
		ecdsaPk, err := crypto.UnmarshalPubkey(key.PublicKey)
		if err != nil {
			panic("Failed to unmarshal public key")
		}
		return crypto.PubkeyToAddress(*ecdsaPk).String()
	case KeyTypeBLSGroupSig:
		return BlsVerKeyToAddress(key.PublicKey).String()
	default:
		return ""
	}
}

func BlsVerKeyToAddress(pubBytes []byte) common.Address {
	return common.BytesToAddress(crypto.Keccak256(pubBytes)[12:])
}

func EcdsaToPublicKey(key *ecdsa.PublicKey) signatures.PublicKey {
	return signatures.PublicKey{
		Type:      KeyTypeSecp256k1,
		PublicKey: crypto.FromECDSAPub(key),
		Id:        crypto.PubkeyToAddress(*key).String(),
	}
}

func BlsKeyToPublicKey(key *bls.VerKey) signatures.PublicKey {
	return signatures.PublicKey{
		Id:        BlsVerKeyToAddress(key.Bytes()).Hex(),
		PublicKey: key.Bytes(),
		Type:      KeyTypeBLSGroupSig,
	}
}

func BlockToHash(block chaintree.Block) ([]byte, error) {
	return ObjToHash(block)
}

func NewEmptyTree(did string, nodeStore nodestore.DagStore) *dag.Dag {
	sw := &safewrap.SafeWrap{}
	treeNode := sw.WrapObject(make(map[string]string))

	chainNode := sw.WrapObject(make(map[string]string))

	root := sw.WrapObject(map[string]interface{}{
		"chain": chainNode.Cid(),
		"tree":  treeNode.Cid(),
		"id":    did,
	})

	// sanity check
	if sw.Err != nil {
		panic(sw.Err)
	}
	dag, err := dag.NewDagWithNodes(context.Background(), nodeStore, root, treeNode, chainNode)
	if err != nil {
		panic(err) // TODO: this err was introduced, keeping external interface the same
	}
	return dag
}

func SignBlock(blockWithHeaders *chaintree.BlockWithHeaders, key *ecdsa.PrivateKey) (*chaintree.BlockWithHeaders, error) {
	hsh, err := BlockToHash(blockWithHeaders.Block)
	if err != nil {
		return nil, fmt.Errorf("error hashing block: %v", err)
	}

	sigBytes, err := crypto.Sign(hsh, key)
	if err != nil {
		return nil, fmt.Errorf("error signing: %v", err)
	}

	addr := crypto.PubkeyToAddress(key.PublicKey).String()

	sig := signatures.Signature{
		Signature: sigBytes,
		Type:      KeyTypeSecp256k1,
	}

	headers := &StandardHeaders{}
	err = typecaster.ToType(blockWithHeaders.Headers, headers)
	if err != nil {
		return nil, fmt.Errorf("error casting headers: %v", err)
	}

	if headers.Signatures == nil {
		headers.Signatures = make(SignatureMap)
	}

	headers.Signatures[addr] = &sig

	var marshaledHeaders map[string]interface{}
	err = typecaster.ToType(headers, &marshaledHeaders)
	if err != nil {
		return nil, fmt.Errorf("error casting headers: %v", err)
	}

	blockWithHeaders.Headers = marshaledHeaders

	return blockWithHeaders, nil
}

func IsBlockSignedBy(blockWithHeaders *chaintree.BlockWithHeaders, addr string) (bool, error) {
	headers := &StandardHeaders{}
	if blockWithHeaders.Headers != nil {
		err := typecaster.ToType(blockWithHeaders.Headers, headers)
		if err != nil {
			return false, fmt.Errorf("error converting headers: %v", err)
		}
	}

	sig, ok := headers.Signatures[addr]
	if !ok {
		log.Error("no signature", "signatures", headers.Signatures)
		return false, nil
	}

	hsh, err := BlockToHash(blockWithHeaders.Block)
	if err != nil {
		log.Error("error wrapping block")
		return false, fmt.Errorf("error wrapping block: %v", err)
	}

	switch sig.Type {
	case KeyTypeSecp256k1:
		ecdsaPubKey, err := crypto.SigToPub(hsh, sig.Signature)
		if err != nil {
			return false, fmt.Errorf("error getting public key: %v", err)
		}
		if crypto.PubkeyToAddress(*ecdsaPubKey).String() != addr {
			return false, fmt.Errorf("unsigned by genesis address %s != %s", crypto.PubkeyToAddress(*ecdsaPubKey).Hex(), addr)
		}
		return crypto.VerifySignature(crypto.FromECDSAPub(ecdsaPubKey), hsh, sig.Signature[:len(sig.Signature)-1]), nil
	}

	log.Error("unknown signature type")
	return false, fmt.Errorf("unkown signature type")
}

func BlsSign(payload interface{}, key *bls.SignKey) (*signatures.Signature, error) {
	hsh, err := ObjToHash(payload)
	if err != nil {
		return nil, fmt.Errorf("error hashing block: %v", err)
	}

	sigBytes, err := key.Sign(hsh)
	if err != nil {
		return nil, fmt.Errorf("error signing: %v", err)
	}

	sig := &signatures.Signature{
		Signature: sigBytes,
		Type:      KeyTypeBLSGroupSig,
	}

	return sig, nil
}

// Sign the bytes sent in with no additional manipulation (no hashing or serializing)
func BlsSignBytes(hsh []byte, key *bls.SignKey) (*signatures.Signature, error) {
	sigBytes, err := key.Sign(hsh)
	if err != nil {
		return nil, fmt.Errorf("error signing: %v", err)
	}

	sig := &signatures.Signature{
		Signature: sigBytes,
		Type:      KeyTypeBLSGroupSig,
	}

	return sig, nil
}

func EcdsaSign(payload interface{}, key *ecdsa.PrivateKey) (*signatures.Signature, error) {
	hsh, err := ObjToHash(payload)
	if err != nil {
		return nil, fmt.Errorf("error hashing block: %v", err)
	}

	sigBytes, err := crypto.Sign(hsh, key)
	if err != nil {
		return nil, fmt.Errorf("error signing: %v", err)
	}
	return &signatures.Signature{
		Signature: sigBytes,
		Type:      KeyTypeSecp256k1,
	}, nil
}

func Verify(hsh []byte, sig signatures.Signature, key signatures.PublicKey) (bool, error) {
	switch sig.Type {
	case KeyTypeSecp256k1:
		recoverdPub, err := crypto.SigToPub(hsh, sig.Signature)
		if err != nil {
			return false, fmt.Errorf("error recovering signature: %v", err)
		}

		if crypto.PubkeyToAddress(*recoverdPub).String() != PublicKeyToAddr(&key) {
			return false, nil
		}

		return crypto.VerifySignature(crypto.FromECDSAPub(recoverdPub), hsh, sig.Signature[:len(sig.Signature)-1]), nil
	case KeyTypeBLSGroupSig:
		verKey := bls.BytesToVerKey(key.PublicKey)
		verified, err := verKey.Verify(sig.Signature, hsh)
		if err != nil {
			log.Error("error verifying", "err", err)
			return false, fmt.Errorf("error verifying: %v", err)
		}
		return verified, nil
	default:
		log.Error("unknown signature type", "type", sig.Type)
		return false, fmt.Errorf("error: unknown signature type: %v", sig.Type)
	}
}

func ObjToHash(payload interface{}) ([]byte, error) {
	sw := &safewrap.SafeWrap{}

	wrapped := sw.WrapObject(payload)
	if sw.Err != nil {
		return nil, fmt.Errorf("error wrapping block: %v", sw.Err)
	}

	hsh := wrapped.Cid().Hash()

	return hsh[2:], nil
}

func MustObjToHash(payload interface{}) []byte {
	hsh, err := ObjToHash(payload)
	if err != nil {
		panic(fmt.Sprintf("error hashing %v", payload))
	}
	return hsh
}

// PassPhraseKey implements a known passphrase -> private Key generator
// following very closely the params from Warp Wallet.
// The only difference here is that the N on scrypt is 256 instead of 218 because
// go requires N to be a power of 2.
// from the Warp Wallet ( https://keybase.io/warp/warp_1.0.9_SHA256_a2067491ab582bde779f4505055807c2479354633a2216b22cf1e92d1a6e4a87.html ):
// s1	=	scrypt(key=(passphrase||0x1), salt=(salt||0x1), N=218, r=8, p=1, dkLen=32)
// s2	=	pbkdf2(key=(passphrase||0x2), salt=(salt||0x2), c=216, dkLen=32, prf=HMAC_SHA256)
// keypair	=	generate_bitcoin_keypair(s1 ⊕ s2)
func PassPhraseKey(passPhrase, salt []byte) (*ecdsa.PrivateKey, error) {
	if len(passPhrase) == 0 {
		return nil, fmt.Errorf("error, must specify a passPhrase")
	}
	var firstSalt, secondSalt []byte
	if len(salt) == 0 {
		firstSalt = []byte{1}
		secondSalt = []byte{2}
	} else {
		hashedSalt := sha256.Sum256(salt)
		firstSalt = hashedSalt[:]
		secondSalt = hashedSalt[:]
	}

	s1, err := scrypt.Key(passPhrase, firstSalt, 256, 8, 1, 32)
	if err != nil {
		return nil, fmt.Errorf("error running scyrpt: %v", err)
	}
	s2 := pbkdf2.Key(passPhrase, secondSalt, 216, 32, sha256.New)
	if err != nil {
		return nil, fmt.Errorf("error running pbkdf2: %v", err)
	}
	dst := make([]byte, 32)
	safeXORBytes(dst, s1, s2, 32)
	return crypto.ToECDSA(dst)
}

// n needs to be smaller or equal than the length of a and b.
func safeXORBytes(dst, a, b []byte, n int) {
	for i := 0; i < n; i++ {
		dst[i] = a[i] ^ b[i]
	}
}
