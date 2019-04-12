package forest

import (
	"bytes"

	"golang.org/x/crypto/openpgp"
)

// IdentityBuilder is a Builder implementation that can create new forest
// nodes without requiring the invocation of an external binary (like gpg
// or another OpenPGP implmenentation).
type IdentityBuilder struct{}

// New builds an Identity node for the user with the given name and metadata, using
// the OpenPGP Entity privkey to define the Identity. That Entity must contain a
// private key with no passphrase.
func (p IdentityBuilder) New(privkey *openpgp.Entity, name *QualifiedContent, metadata *QualifiedContent) (*Identity, error) {
	// make an empty identity and populate all fields that need to be known before
	// signing the data
	identity := newIdentity()
	identity.Version = Version
	identity.Type = NodeTypeIdentity
	identity.Parent = NullHash()
	identity.Depth = 0
	identity.Name = *name
	identity.Metadata = *metadata
	keybuf := new(bytes.Buffer)
	// serialize the Entity (without the private key) into the node. This will just
	// be the public key and metadata
	if err := privkey.Serialize(keybuf); err != nil {
		return nil, err
	}
	qKey, err := NewQualifiedKey(KeyTypeOpenPGP, keybuf.Bytes())
	if err != nil {
		return nil, err
	}
	identity.PublicKey = *qKey
	identity.SignatureAuthority = NullHash()
	idDesc, err := NewHashDescriptor(HashTypeSHA512_256, int(HashDigestLengthSHA512_256))
	if err != nil {
		return nil, err
	}
	identity.IDDesc = *idDesc

	// we've defined all pre-signature fields, it's time to sign the data
	signedData := new(bytes.Buffer)
	if err := identity.WriteDataForSigningInto(signedData); err != nil {
		return nil, err
	}
	signature := new(bytes.Buffer)
	if err := openpgp.DetachSign(signature, privkey, signedData, nil); err != nil {
		return nil, err
	}
	qs, err := NewQualifiedSignature(SignatureTypeOpenPGP, signature.Bytes())
	if err != nil {
		return nil, err
	}
	identity.Signature = *qs

	// determine the node's final hash ID
	id, err := identity.computeID()
	if err != nil {
		return nil, err
	}
	identity.id = Value(id)

	return identity, nil
}
