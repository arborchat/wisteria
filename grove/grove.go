/*
Package grove implements an on-disk storage format for arbor forest
nodes. This hierarchical storage format is called a "grove", and
the management type implemented by this package satisfies the
forest.Store interface.

Note: this package is not yet complete.
*/
package grove

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"git.sr.ht/~whereswaldon/forest-go"
	"git.sr.ht/~whereswaldon/forest-go/fields"
)

// File represents a type that supports file-like operations. *os.File
// implements this interface, and will likely be used most of the time.
// This interface exists mostly to simply testing.
type File interface {
	io.ReadWriteCloser
	Name() string
}

// FS represents a type that acts as a filesystem. It can create and
// open files at specific paths
type FS interface {
	Open(path string) (File, error)
	Create(path string) (File, error)
	OpenFile(path string, flag int, perm os.FileMode) (File, error)
}

// RelativeFS is a file system that acts relative to a specific path
type RelativeFS struct {
	Root string
}

// ensure RelativeFS satisfies the FS interface
var _ FS = RelativeFS{}

func (r RelativeFS) resolve(path string) string {
	return filepath.Join(r.Root, path)
}

// Open opens the given path as an absolute path relative to the root
// of the RelativeFS
func (r RelativeFS) Open(path string) (File, error) {
	return os.Open(r.resolve(path))
}

// Create makes the given path as an absolute path relative to the root
// of the RelativeFS
func (r RelativeFS) Create(path string) (File, error) {
	return os.Create(r.resolve(path))
}

// OpenFile opens the given path as an absolute path relative to the root
// of the RelativeFS
func (r RelativeFS) OpenFile(path string, flag int, perm os.FileMode) (File, error) {
	return os.OpenFile(r.resolve(path), flag, perm)
}

// Grove is an on-disk store for arbor forest nodes.
type Grove struct {
	FS
}

// New constructs a Grove that stores nodes in a hierarchy rooted at
// the given path.
func New(root string) (*Grove, error) {
	return NewWithFS(RelativeFS{root})
}

// NewWithFS constructs a Grove using the given FS implementation to
// access its nodes. This is primarily useful for testing.
func NewWithFS(fs FS) (*Grove, error) {
	if fs == nil {
		return nil, fmt.Errorf("fs cannot be nil")
	}
	return &Grove{
		FS: fs,
	}, nil
}

// Get searches the grove for a node with the given id. It returns the node if it was
// found, a boolean indicating whether it was found, and an error (if there was a
// problem searching for the node).
// The returned `present` will never be true unless the returned `node` holds an
// actual node struct. If the file holding a node exists on disk but was unable
// to be opened, read, or parsed, `present` will still be false.
func (g *Grove) Get(nodeID *fields.QualifiedHash) (node forest.Node, present bool, err error) {
	filename := nodeID.String()
	file, err := g.Open(filename)
	// if the file doesn't exist, just return false with no error
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	// if it's some other error, wrap it and return
	if err != nil {
		return nil, false, fmt.Errorf("failed opening node file \"%s\": %w", filename, err)
	}
	b, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, false, fmt.Errorf("failed reading bytes from \"%s\": %w", filename, err)
	}
	node, err = forest.UnmarshalBinaryNode(b)
	if err != nil {
		return nil, false, fmt.Errorf("failed unmarshalling node from \"%s\": %w", filename, err)
	}
	return node, true, nil
}

// GetIdentity returns an Identity node with the given ID (if it is present
// in the grove). This operation may be faster than using Get, as the grove
// may be able to do less search work when it knows the type of node you're
// looking for in advance.
//
// BUG(whereswaldon): The current implementation may return nodes of the
// wrong NodeType if they match the provided ID
func (g *Grove) GetIdentity(id *fields.QualifiedHash) (forest.Node, bool, error) {
	// this naiive implementation is not efficient, but works as a short-term
	// thing.
	//
	// TODO: change the on-disk representation so that operations like this can
	// be fast (store different node types in different directories, etc...)
	return g.Get(id)
}

// GetCommunity returns an Community node with the given ID (if it is present
// in the grove). This operation may be faster than using Get, as the grove
// may be able to do less search work when it knows the type of node you're
// looking for in advance.
//
// BUG(whereswaldon): The current implementation may return nodes of the
// wrong NodeType if they match the provided ID
func (g *Grove) GetCommunity(id *fields.QualifiedHash) (forest.Node, bool, error) {
	// this naiive implementation is not efficient, but works as a short-term
	// thing.
	//
	// TODO: change the on-disk representation so that operations like this can
	// be fast (store different node types in different directories, etc...)
	return g.Get(id)
}

// GetConversation returns an Conversation node with the given ID (if it is present
// in the grove). This operation may be faster than using Get, as the grove
// may be able to do less search work when it knows the type of node you're
// looking for and its parent node in advance.
//
// BUG(whereswaldon): The current implementation may return nodes of the
// wrong NodeType if they match the provided ID
func (g *Grove) GetConversation(communityID, conversationID *fields.QualifiedHash) (forest.Node, bool, error) {
	// this naiive implementation is not efficient, but works as a short-term
	// thing.
	//
	// TODO: change the on-disk representation so that operations like this can
	// be fast (store different node types in different directories, etc...)
	return g.Get(conversationID)
}

// GetReply returns an Reply node with the given ID (if it is present
// in the grove). This operation may be faster than using Get, as the grove
// may be able to do less search work when it knows the type of node you're
// looking for and its parent community and conversation node in advance.
//
// BUG(whereswaldon): The current implementation may return nodes of the
// wrong NodeType if they match the provided ID
func (g *Grove) GetReply(communityID, conversationID, replyID *fields.QualifiedHash) (forest.Node, bool, error) {
	// this naiive implementation is not efficient, but works as a short-term
	// thing.
	//
	// TODO: change the on-disk representation so that operations like this can
	// be fast (store different node types in different directories, etc...)
	return g.Get(replyID)
}

// CopyInto copies all nodes from the store into the provided store.
//
// BUG(whereswaldon): this method is not yet implemented. It requires
// more extensive file manipulation than other Grove methods (listing
// directory contents) and has therefore been deprioritized in favor
// of the functionality that can be implemented simply. However, it is
// implementable, and should be done as soon as is feasible.
func (g *Grove) CopyInto(other forest.Store) error {
	return fmt.Errorf("method CopyInto() is not currently implemented on Grove")
}
