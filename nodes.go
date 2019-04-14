package forest

import (
	"bytes"
	"crypto/sha512"
	"encoding"
	"fmt"
	"hash"
	"io"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/packet"
)

// generic node
type Node struct {
	// the ID is deterministically computed from the rest of the values
	id                 Value
	Type               NodeType
	SchemaVersion      Version
	Parent             QualifiedHash
	IDDesc             HashDescriptor
	Depth              TreeDepth
	Metadata           QualifiedContent
	SignatureAuthority QualifiedHash
	Signature          QualifiedSignature
	// WriteNodeTypeFieldsInto allows higher-level logic to define
	// how to serialize extra fields. See the concrete Node type
	// implementations for details
	WriteNodeTypeFieldsInto func(w io.Writer) error
}

func MarshalAllInto(w io.Writer, marshalers ...encoding.BinaryMarshaler) error {
	for _, marshaler := range marshalers {
		b, err := marshaler.MarshalBinary()
		if err != nil {
			return err
		}
		_, err = w.Write(b)
		if err != nil {
			return err
		}
	}
	return nil
}

// computeID determines the correct value of this node's ID without modifying
// the node.
func (n Node) computeID() ([]byte, error) {
	// map from HashType to the function that creates an instance of that hash
	// algorithm
	hashType2Func := map[HashType]func() hash.Hash{
		HashTypeSHA512_256: sha512.New512_256,
	}
	if HashType(n.IDDesc.Type) == HashTypeNullHash {
		return []byte{}, nil
	}
	binaryContent, err := n.MarshalBinary()
	if err != nil {
		return nil, err
	}
	hashFunc, found := hashType2Func[HashType(n.IDDesc.Type)]
	if !found {
		return nil, fmt.Errorf("Unknown HashType %d", n.IDDesc.Type)
	}
	hasher := hashFunc()
	_, _ = hasher.Write(binaryContent) // never errors
	return hasher.Sum(nil), nil
}

// ValidateID returns whether the ID of this Node matches the data. The first
// return value indicates the result of the comparison. If there is an error,
// the first return value will always be false and the second will indicate
// what went wrong when computing the hash.
func (n Node) ValidateID() (bool, error) {
	currentID := n.ID()
	id, err := n.computeID()
	if err != nil {
		return false, err
	}
	computedID := QualifiedHash{
		Descriptor: descriptor(n.IDDesc),
		Value:      Value(id),
	}
	return qualified(currentID).Equals(qualified(computedID)), nil
}

// ValidateSignature returns whether the signature contained in this Node is a valid
// signature for the given Identity. When validating an Identity node, you should
// pass the Identity to this method.
func (n Node) ValidateSignatureFor(identity *Identity) (bool, error) {
	if qualified(n.SignatureAuthority).Equals(qualified(NullHash())) {
		if n.Type != NodeTypeIdentity {
			return false, fmt.Errorf("Only Identity nodes can have the null hash as their Signature Authority")
		}
	} else if !qualified(n.SignatureAuthority).Equals(qualified(identity.ID())) {
		return false, fmt.Errorf("This node was signed by a different identity")
	}
	// get the key used to sign this node
	pubkeyBuf := bytes.NewBuffer([]byte(identity.PublicKey.Value))
	pubkeyEntity, err := openpgp.ReadEntity(packet.NewReader(pubkeyBuf))
	if err != nil {
		return false, err
	}

	signedContentBuf := new(bytes.Buffer)
	if err = n.WriteDataForSigningInto(signedContentBuf); err != nil {
		return false, err
	}
	signatureBuf := bytes.NewBuffer([]byte(n.Signature.Value))
	keyring := openpgp.EntityList([]*openpgp.Entity{pubkeyEntity})
	_, err = openpgp.CheckDetachedSignature(keyring, signedContentBuf, signatureBuf)
	if err != nil {
		return false, err
	}
	return true, nil
}

// Compute and return the Node's ID as a Qualified Hash
func (n Node) ID() QualifiedHash {
	return QualifiedHash{
		Descriptor: descriptor(n.IDDesc),
		Value:      n.id,
	}
}

func (n Node) WriteCommonFieldsInto(w io.Writer) error {
	// this slice defines the order in which the fields are written
	return MarshalAllInto(w, n.presignSerializationOrder()...)
}

type BidirectionalBinaryMarshaler interface {
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
}

func (n *Node) serializationOrder() []BidirectionalBinaryMarshaler {
	return []BidirectionalBinaryMarshaler{
		&n.SchemaVersion,
		&n.Type,
		&n.Parent,
		&n.IDDesc,
		&n.Depth,
		&n.Metadata,
		&n.SignatureAuthority,
		&n.Signature,
	}
}

func asMarshaler(in []BidirectionalBinaryMarshaler) []encoding.BinaryMarshaler {
	out := make([]encoding.BinaryMarshaler, len(in))
	for i, f := range in {
		out[i] = encoding.BinaryMarshaler(f)
	}
	return out
}

func asUnmarshaler(in []BidirectionalBinaryMarshaler) []encoding.BinaryUnmarshaler {
	out := make([]encoding.BinaryUnmarshaler, len(in))
	for i, f := range in {
		out[i] = encoding.BinaryUnmarshaler(f)
	}
	return out
}

func (n *Node) presignSerializationOrder() []encoding.BinaryMarshaler {
	fields := n.serializationOrder()
	fields = fields[:len(fields)-1] // drop the signature
	return asMarshaler(fields)
}

func (n *Node) postsignSerializationOrder() []encoding.BinaryMarshaler {
	fields := n.serializationOrder()
	return asMarshaler(fields[len(fields)-1:]) // drop the signature
}

func (n Node) WriteSignatureInto(w io.Writer) error {
	return MarshalAllInto(w, n.postsignSerializationOrder()...)
}

func (n Node) WriteDataForSigningInto(w io.Writer) error {
	if err := n.WriteCommonFieldsInto(w); err != nil {
		return err
	}
	if err := n.WriteNodeTypeFieldsInto(w); err != nil {
		return err
	}
	return nil
}

func (n Node) MarshalBinary() ([]byte, error) {
	// this is a template method. It always writes the common fields,
	// then invokes a method responsible for writing data that varies
	// between Node Types, then writes the final data
	b := new(bytes.Buffer)
	writeFuncs := []func(io.Writer) error{
		n.WriteDataForSigningInto,
		n.WriteSignatureInto,
	}

	// invoke the methods in the order defined by the slice above
	for _, f := range writeFuncs {
		err := f(b)
		if err != nil {
			return nil, err
		}
	}
	// invoke the methods in the order defined by the slice above	}
	return b.Bytes(), nil
}

// concrete nodes
type Identity struct {
	Node
	Name      QualifiedContent
	PublicKey QualifiedKey
}

func newIdentity() *Identity {
	i := new(Identity)
	// define how to serialize this node type's fields
	i.Node.WriteNodeTypeFieldsInto = func(w io.Writer) error {
		return MarshalAllInto(w, i.Name, i.PublicKey)
	}
	return i
}

type Community struct {
	Node
	Name QualifiedContent
}

func newCommunity() *Community {
	c := new(Community)
	// define how to serialize this node type's fields
	c.Node.WriteNodeTypeFieldsInto = func(w io.Writer) error {
		return MarshalAllInto(w, c.Name)
	}
	return c
}

type Conversation struct {
	Node
	Content QualifiedContent
}

func newConversation() *Conversation {
	c := new(Conversation)
	// define how to serialize this node type's fields
	c.Node.WriteNodeTypeFieldsInto = func(w io.Writer) error {
		return MarshalAllInto(w, c.Content)
	}
	return c
}

type Reply struct {
	Node
	ConversationID QualifiedHash
	Content        QualifiedContent
}

func newReply() *Reply {
	r := new(Reply)
	// define how to serialize this node type's fields
	r.Node.WriteNodeTypeFieldsInto = func(w io.Writer) error {
		return MarshalAllInto(w, r.ConversationID, r.Content)
	}
	return r
}
