package forest

import (
	"bytes"
	"fmt"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/packet"
)

type SignatureValidator interface {
	MarshalSignedData() ([]byte, error)
	Signature() *QualifiedSignature
	SignatureIdentityHash() *QualifiedHash
	IsIdentity() bool
}

// ValidateSignature returns whether the signature contained in this SignatureValidator is a valid
// signature for the given Identity. When validating an Identity node, you should
// pass the same Identity as the second parameter.
func ValidateSignature(v SignatureValidator, identity *Identity) (bool, error) {
	sigIdHash := v.SignatureIdentityHash()
	if sigIdHash.Equals(NullHash()) {
		if !v.IsIdentity() {
			return false, fmt.Errorf("Only Identity nodes can have the null hash as their Signature Authority")
		}
	} else if !sigIdHash.Equals(identity.ID()) {
		return false, fmt.Errorf("This node was signed by a different identity")
	}
	// get the key used to sign this node
	pubkeyBuf := bytes.NewBuffer([]byte(identity.PublicKey.Value))
	pubkeyEntity, err := openpgp.ReadEntity(packet.NewReader(pubkeyBuf))
	if err != nil {
		return false, err
	}

	signedContent, err := v.MarshalSignedData()
	if err != nil {
		return false, err
	}
	signedContentBuf := bytes.NewBuffer(signedContent)

	signatureBuf := bytes.NewBuffer([]byte(v.Signature().Value))
	keyring := openpgp.EntityList([]*openpgp.Entity{pubkeyEntity})
	_, err = openpgp.CheckDetachedSignature(keyring, signedContentBuf, signatureBuf)
	if err != nil {
		return false, err
	}
	return true, nil
}
