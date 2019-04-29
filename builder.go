package forest

import (
	"bytes"
	"fmt"

	"git.sr.ht/~whereswaldon/forest-go/fields"
	"golang.org/x/crypto/openpgp"
)

// IdentityBuilder is a Builder implementation that can create new forest
// nodes without requiring the invocation of an external binary (like gpg
// or another OpenPGP implmenentation).
type IdentityBuilder struct{}

// New builds an Identity node for the user with the given name and metadata, using
// the OpenPGP Entity privkey to define the Identity. That Entity must contain a
// private key with no passphrase.
func (p IdentityBuilder) New(privkey *openpgp.Entity, name *fields.QualifiedContent, metadata *fields.QualifiedContent) (*Identity, error) {
	// make an empty identity and populate all fields that need to be known before
	// signing the data
	identity := newIdentity()
	identity.SchemaVersion = fields.CurrentVersion
	identity.Type = fields.NodeTypeIdentity
	identity.Parent = *fields.NullHash()
	identity.Depth = 0
	identity.Name = *name
	identity.Metadata = *metadata
	keybuf := new(bytes.Buffer)
	// serialize the Entity (without the private key) into the node. This will just
	// be the public key and metadata
	if err := privkey.Serialize(keybuf); err != nil {
		return nil, err
	}
	qKey, err := fields.NewQualifiedKey(fields.KeyTypeOpenPGP, keybuf.Bytes())
	if err != nil {
		return nil, err
	}
	identity.PublicKey = *qKey
	identity.SignatureAuthority = *fields.NullHash()
	idDesc, err := fields.NewHashDescriptor(fields.HashTypeSHA512_256, int(fields.HashDigestLengthSHA512_256))
	if err != nil {
		return nil, err
	}
	identity.IDDesc = *idDesc

	// we've defined all pre-signature fields, it's time to sign the data
	signedDataBytes, err := identity.MarshalSignedData()
	if err != nil {
		return nil, err
	}
	signedData := bytes.NewBuffer(signedDataBytes)
	signature := new(bytes.Buffer)
	if err := openpgp.DetachSign(signature, privkey, signedData, nil); err != nil {
		return nil, err
	}
	qs, err := fields.NewQualifiedSignature(fields.SignatureTypeOpenPGP, signature.Bytes())
	if err != nil {
		return nil, err
	}
	identity.commonNode.Signature = *qs

	// determine the node's final hash ID
	id, err := computeID(identity)
	if err != nil {
		return nil, err
	}
	identity.id = fields.Value(id)

	return identity, nil
}

// NewCommunity creates a community node (signed by the given identity with the given privkey).
func NewCommunity(identity *Identity, privkey *openpgp.Entity, name *fields.QualifiedContent, metadata *fields.QualifiedContent) (*Community, error) {
	c := newCommunity()
	c.SchemaVersion = fields.CurrentVersion
	c.Type = fields.NodeTypeCommunity
	c.Parent = *fields.NullHash()
	c.Depth = 0
	c.Name = *name
	c.Metadata = *metadata
	c.SignatureAuthority = *identity.ID()
	idDesc, err := fields.NewHashDescriptor(fields.HashTypeSHA512_256, int(fields.HashDigestLengthSHA512_256))
	if err != nil {
		return nil, err
	}
	c.IDDesc = *idDesc

	// we've defined all pre-signature fields, it's time to sign the data
	signedDataBytes, err := c.MarshalSignedData()
	if err != nil {
		return nil, err
	}
	signedData := bytes.NewBuffer(signedDataBytes)
	signature := new(bytes.Buffer)
	if err := openpgp.DetachSign(signature, privkey, signedData, nil); err != nil {
		return nil, err
	}
	qs, err := fields.NewQualifiedSignature(fields.SignatureTypeOpenPGP, signature.Bytes())
	if err != nil {
		return nil, err
	}
	c.commonNode.Signature = *qs

	// determine the node's final hash ID
	id, err := computeID(c)
	if err != nil {
		return nil, err
	}
	c.id = fields.Value(id)

	return c, nil
}

// NewConversation creates a conversation node (signed by the given identity with the given privkey) as a child of the given community
func NewConversation(identity *Identity, privkey *openpgp.Entity, parent *Community, content *fields.QualifiedContent, metadata *fields.QualifiedContent) (*Conversation, error) {
	c := newConversation()
	c.SchemaVersion = fields.CurrentVersion
	c.Type = fields.NodeTypeConversation
	c.Parent = *parent.ID()
	c.Depth = parent.Depth + 1
	c.Content = *content
	c.Metadata = *metadata
	c.SignatureAuthority = *identity.ID()
	idDesc, err := fields.NewHashDescriptor(fields.HashTypeSHA512_256, int(fields.HashDigestLengthSHA512_256))
	if err != nil {
		return nil, err
	}
	c.IDDesc = *idDesc

	// we've defined all pre-signature fields, it's time to sign the data
	signedDataBytes, err := c.MarshalSignedData()
	if err != nil {
		return nil, err
	}
	signedData := bytes.NewBuffer(signedDataBytes)
	signature := new(bytes.Buffer)
	if err := openpgp.DetachSign(signature, privkey, signedData, nil); err != nil {
		return nil, err
	}
	qs, err := fields.NewQualifiedSignature(fields.SignatureTypeOpenPGP, signature.Bytes())
	if err != nil {
		return nil, err
	}
	c.commonNode.Signature = *qs

	// determine the node's final hash ID
	id, err := computeID(c)
	if err != nil {
		return nil, err
	}
	c.id = fields.Value(id)

	return c, nil
}

// NewReply creates a conversation node (signed by the given identity with the given privkey) as a child of the given conversation
func NewReply(identity *Identity, privkey *openpgp.Entity, parent interface{}, content *fields.QualifiedContent, metadata *fields.QualifiedContent) (*Reply, error) {
	r := newReply()
	r.SchemaVersion = fields.CurrentVersion
	r.Type = fields.NodeTypeReply
	switch concreteParent := parent.(type) {
	case *Conversation:
		r.CommunityID = concreteParent.Parent
		r.Parent = *concreteParent.ID()
		r.Depth = concreteParent.Depth + 1
	case *Reply:
		r.CommunityID = concreteParent.CommunityID
		r.Parent = *concreteParent.ID()
		r.Depth = concreteParent.Depth + 1
	default:
		return nil, fmt.Errorf("parent must be either a conversation or reply node")

	}
	r.Content = *content
	r.Metadata = *metadata
	r.SignatureAuthority = *identity.ID()
	idDesc, err := fields.NewHashDescriptor(fields.HashTypeSHA512_256, int(fields.HashDigestLengthSHA512_256))
	if err != nil {
		return nil, err
	}
	r.IDDesc = *idDesc

	// we've defined all pre-signature fields, it's time to sign the data
	signedDataBytes, err := r.MarshalSignedData()
	if err != nil {
		return nil, err
	}
	signedData := bytes.NewBuffer(signedDataBytes)
	signature := new(bytes.Buffer)
	if err := openpgp.DetachSign(signature, privkey, signedData, nil); err != nil {
		return nil, err
	}
	qs, err := fields.NewQualifiedSignature(fields.SignatureTypeOpenPGP, signature.Bytes())
	if err != nil {
		return nil, err
	}
	r.commonNode.Signature = *qs

	// determine the node's final hash ID
	id, err := computeID(r)
	if err != nil {
		return nil, err
	}
	r.id = fields.Value(id)

	return r, nil
}
