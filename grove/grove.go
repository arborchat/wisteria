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
	"strings"

	"git.sr.ht/~whereswaldon/forest-go"
	"git.sr.ht/~whereswaldon/forest-go/fields"
)

// File represents a type that supports file-like operations. *os.File
// implements this interface, and will likely be used most of the time.
// This interface exists mostly to simply testing.
type File interface {
	io.ReadWriteCloser
	Name() string
	Readdir(n int) ([]os.FileInfo, error)
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
func (g *Grove) Get(nodeID *fields.QualifiedHash) (forest.Node, bool, error) {
	filename, err := nodeID.MarshalString()
	if err != nil {
		return nil, false, fmt.Errorf("failed determining file name for node: %w", err)
	}
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
	node, err := forest.UnmarshalBinaryNode(b)
	if err != nil {
		return nil, false, fmt.Errorf("failed unmarshalling node from \"%s\": %w", filename, err)
	}
	return node, true, nil
}

// Children returns the IDs of all known child nodes of the specified ID.
// Any error opening, reading, or parsing files in the grove that occurs
// during the search for child nodes will cause the entire operation to
// error.
func (g *Grove) Children(id *fields.QualifiedHash) ([]*fields.QualifiedHash, error) {
	// open root of grove hierarchy so we can list its nodes
	rootDir, err := g.Open("")
	if err != nil {
		return nil, fmt.Errorf("failed opening grove root dir: %w", err)
	}
	info, err := rootDir.Readdir(-1) // read whole directory at once. Inefficient
	if err != nil {
		return nil, fmt.Errorf("failed listing files in grove: %w", err)
	}
	nodeInfo := make([]os.FileInfo, 0, len(info))
	// find all files that are plausibly nodes
	for _, fileInfo := range info {
		// search for the string form of all supported hash types
		for _, hashName := range fields.HashNames {
			if strings.HasPrefix(fileInfo.Name(), hashName) {
				nodeInfo = append(nodeInfo, fileInfo)
			}
		}
	}
	children := make([]*fields.QualifiedHash, 0, len(nodeInfo))
	for _, nodeFileInfo := range nodeInfo {
		nodeFile, err := g.Open(nodeFileInfo.Name())
		if err != nil {
			return nil, fmt.Errorf("failed opening node file %s: %w", nodeFileInfo.Name(), err)
		}
		nodeData, err := ioutil.ReadAll(nodeFile)
		if err != nil {
			return nil, fmt.Errorf("failed reading node file %s: %w", nodeFileInfo.Name(), err)
		}
		node, err := forest.UnmarshalBinaryNode(nodeData)
		if err != nil {
			return nil, fmt.Errorf("failed parsing node file %s: %w", nodeFileInfo.Name(), err)
		}
		if node.ParentID().Equals(id) {
			children = append(children, node.ID())
		}
	}

	return children, nil
}
