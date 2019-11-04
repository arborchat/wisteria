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
	"sort"
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

// getAllNodeFileInfo returns a slice of information about all node files
// within the grove.
func (g *Grove) getAllNodeFileInfo() ([]os.FileInfo, error) {
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
	return nodeInfo, nil
}

// nodeFromInfo converts the info about a file into a node extracted from
// the contents of that file (it opens, reads, and parses the file).
func (g *Grove) nodeFromInfo(info os.FileInfo) (forest.Node, error) {
	nodeFile, err := g.Open(info.Name())
	if err != nil {
		return nil, fmt.Errorf("failed opening node file %s: %w", info.Name(), err)
	}
	nodeData, err := ioutil.ReadAll(nodeFile)
	if err != nil {
		return nil, fmt.Errorf("failed reading node file %s: %w", info.Name(), err)
	}
	node, err := forest.UnmarshalBinaryNode(nodeData)
	if err != nil {
		return nil, fmt.Errorf("failed parsing node file %s: %w", info.Name(), err)
	}
	return node, nil
}

// nodesFromInfo batch-converts a slice of file info into a slice of
// forest nodes by calling nodeFromInfo on each.
func (g *Grove) nodesFromInfo(info []os.FileInfo) ([]forest.Node, error) {
	nodes := make([]forest.Node, 0, len(info))
	for _, nodeFileInfo := range info {
		node, err := g.nodeFromInfo(nodeFileInfo)
		if err != nil {
			return nil, fmt.Errorf("failed transforming fileInfo into Node: %w", err)
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

// allNodes returns a slice of every node in the grove.
func (g *Grove) allNodes() ([]forest.Node, error) {
	nodeInfo, err := g.getAllNodeFileInfo()
	if err != nil {
		return nil, fmt.Errorf("failed listing node file candidates: %w", err)
	}
	nodes, err := g.nodesFromInfo(nodeInfo)
	if err != nil {
		return nil, fmt.Errorf("failed converting node files into nodes: %w", err)
	}
	return nodes, nil
}

// Children returns the IDs of all known child nodes of the specified ID.
// Any error opening, reading, or parsing files in the grove that occurs
// during the search for child nodes will cause the entire operation to
// error.
func (g *Grove) Children(id *fields.QualifiedHash) ([]*fields.QualifiedHash, error) {
	nodes, err := g.allNodes()
	if err != nil {
		return nil, fmt.Errorf("failed getting all nodes from grove: %w", err)
	}
	children := make([]*fields.QualifiedHash, 0, len(nodes))
	for _, node := range nodes {
		if node.ParentID().Equals(id) {
			children = append(children, node.ID())
		}
	}

	return children, nil
}

// Recent returns a slice of the most recently-created nodes of the given type.
func (g *Grove) Recent(nodeType fields.NodeType, quantity int) ([]forest.Node, error) {
	nodes, err := g.allNodes()
	if err != nil {
		return nil, fmt.Errorf("failed getting all nodes from grove: %w", err)
	}
	// TODO: find a cleaner way to sort nodes by time
	sort.Slice(nodes, func(i, j int) bool {
		var a, b forest.CommonNode
		switch n := nodes[i].(type) {
		case *forest.Identity:
			a = n.CommonNode
		case *forest.Community:
			a = n.CommonNode
		case *forest.Reply:
			a = n.CommonNode
		}
		switch n := nodes[j].(type) {
		case *forest.Identity:
			b = n.CommonNode
		case *forest.Community:
			b = n.CommonNode
		case *forest.Reply:
			b = n.CommonNode
		}
		return a.Created > b.Created
	})
	rightType := make([]forest.Node, 0, quantity)
	for _, node := range nodes {
		switch node.(type) {
		case *forest.Identity:
			if nodeType == fields.NodeTypeIdentity {
				rightType = append(rightType, node)
			}
		case *forest.Community:
			if nodeType == fields.NodeTypeCommunity {
				rightType = append(rightType, node)
			}
		case *forest.Reply:
			if nodeType == fields.NodeTypeReply {
				rightType = append(rightType, node)
			}
		}
	}
	if len(rightType) > quantity {
		rightType = rightType[:quantity]
	}
	return rightType, nil
}
