package grove_test

import (
	"bytes"
	"os"
	"testing"

	forest "git.sr.ht/~whereswaldon/forest-go"
	"git.sr.ht/~whereswaldon/forest-go/grove"
	"git.sr.ht/~whereswaldon/forest-go/testkeys"
)

// fakeFile implements the grove.File interface, but is entirely in-memory.
// This helps speed testing.
type fakeFile struct {
	*bytes.Buffer
	name string
}

var _ grove.File = &fakeFile{}

func newFakeFile(name string, content []byte) *fakeFile {
	return &fakeFile{
		name:   name,
		Buffer: bytes.NewBuffer(content),
	}
}

func (f *fakeFile) Name() string {
	return f.name
}

// needed to implement Close so that fakeFile is a io.ReadWriteCloser
func (f *fakeFile) Close() error {
	return nil
}

// errFile implements the grove.File interface and wraps another grove.File.
// If the errFile's error field is set to nil, it is a transparent wrapper
// for the underlying File. If the field is set to a non-nil error value,
// this will be returned from all operations that can return an error.
type errFile struct {
	error
	wrappedFile grove.File
}

var _ grove.File = &errFile{}

func NewErrFile(file grove.File) *errFile {
	return &errFile{
		wrappedFile: file,
	}
}

func (e *errFile) Name() string {
	return e.wrappedFile.Name()
}

func (e *errFile) Read(b []byte) (int, error) {
	if e.error != nil {
		return 0, e.error
	}
	return e.wrappedFile.Read(b)
}

func (e *errFile) Write(b []byte) (int, error) {
	if e.error != nil {
		return 0, e.error
	}
	return e.wrappedFile.Write(b)
}

func (e *errFile) Close() error {
	if e.error != nil {
		return e.error
	}
	return e.wrappedFile.Close()
}

// fakeFS implements grove.FS, but is entirely in-memory.
type fakeFS struct {
	files map[string]grove.File
}

var _ grove.FS = fakeFS{}

func newFakeFS() fakeFS {
	return fakeFS{
		make(map[string]grove.File),
	}
}

// Open opens the given path as an absolute path relative to the root
// of the fakeFS
func (r fakeFS) Open(path string) (grove.File, error) {
	file, exists := r.files[path]
	if !exists {
		return nil, os.ErrNotExist
	}
	return file, nil
}

// Create makes the given path as an absolute path relative to the root
// of the fakeFS
func (r fakeFS) Create(path string) (grove.File, error) {
	// mimic os.Create(), so creating a file that already exists truncates
	// the current one
	file := newFakeFile(path, []byte{})
	r.files[path] = file

	return file, nil
}

// OpenFile opens the given path as an absolute path relative to the root
// of the fakeFS
func (r fakeFS) OpenFile(path string, flag int, perm os.FileMode) (grove.File, error) {
	return r.Open(path)
}

// errFS is a testing type that wraps an ordinary FS with the ability to
// return a specific error on any function call.
type errFS struct {
	fs grove.FS
	error
}

var _ grove.FS = errFS{}

func newErrFS(fs grove.FS) *errFS {
	return &errFS{
		fs: fs,
	}
}

// Open opens the given path as an absolute path relative to the root
// of the errFS
func (r errFS) Open(path string) (grove.File, error) {
	if r.error != nil {
		return nil, r.error
	}
	return r.fs.Open(path)
}

// Create makes the given path as an absolute path relative to the root
// of the errFS
func (r errFS) Create(path string) (grove.File, error) {
	if r.error != nil {
		return nil, r.error
	}
	return r.fs.Create(path)
}

// OpenFile opens the given path as an absolute path relative to the root
// of the errFS
func (r errFS) OpenFile(path string, flag int, perm os.FileMode) (grove.File, error) {
	if r.error != nil {
		return nil, r.error
	}
	return r.fs.OpenFile(path, flag, perm)
}

type testNodeBuilder struct {
	*testing.T
	*forest.Builder
	*forest.Community
}

func NewNodeBuilder(t *testing.T) *testNodeBuilder {
	signer := testkeys.Signer(t, testkeys.PrivKey1)
	id, err := forest.NewIdentity(signer, "node-builder", "")
	if err != nil {
		t.Errorf("Failed to create identity: %v", err)
		return nil
	}
	builder := forest.As(id, signer)
	community, err := builder.NewCommunity("nodes-built-for-testing", "")
	if err != nil {
		t.Errorf("Failed to create community: %v", err)
		return nil
	}
	return &testNodeBuilder{
		T:         t,
		Builder:   builder,
		Community: community,
	}
}

// newReplyFile creates a fakeFile that contains the binary data for a reply
// node that is a direct child of the given community and constructed by the
// given builder. It returns the reply node as a convenience for testing.
func (tnb *testNodeBuilder) newReplyFile(content string) (*forest.Reply, *fakeFile) {
	reply, err := tnb.NewReply(tnb.Community, content, "")
	if err != nil {
		tnb.T.Errorf("Failed generating test reply node: %v", err)
	}
	b, err := reply.MarshalBinary()
	if err != nil {
		tnb.T.Errorf("Failed marshalling test reply node: %v", err)
	}
	id := reply.ID()
	nodeID, err := id.MarshalString()
	if err != nil {
		tnb.T.Errorf("Failed to marshal node id: %v", err)
	}
	return reply, newFakeFile(nodeID, b)
}

func TestCreateEmptyGrove(t *testing.T) {
	fs := newFakeFS()
	grove, err := grove.NewWithFS(fs)
	if err != nil {
		t.Fatalf("Failed to create grove with fake fs: %v", err)
	}
	if grove == nil {
		t.Fatalf("Grove constructor did not err, but returned nil grove")
	}
}

func TestCreateGroveFromNil(t *testing.T) {
	_, err := grove.NewWithFS(nil)
	if err == nil {
		t.Fatalf("Created grove with nil fs, should have errored")
	}
}

func TestGroveGet(t *testing.T) {
	fs := newFakeFS()
	fakeNodeBuilder := NewNodeBuilder(t)
	reply, replyFile := fakeNodeBuilder.newReplyFile("test content")
	g, err := grove.NewWithFS(fs)
	if err != nil {
		t.Errorf("Failed constructing grove: %v", err)
	}

	// no nodes in fs, make sure we get nothing
	if node, present, err := g.Get(reply.ID()); err != nil {
		t.Errorf("Failed looking for %v (not present): %v", reply.ID(), err)
	} else if present {
		t.Errorf("Grove indicated that a node was present when it was not added")
	} else if node != nil {
		t.Errorf("Grove returned a node when the requested node was not present")
	}

	// add node to fs, now should be discoverable
	fs.files[replyFile.Name()] = replyFile

	// no nodes in fs, make sure we get nothing
	if node, present, err := g.Get(reply.ID()); err != nil {
		t.Errorf("Failed looking for %v (present): %v", reply.ID(), err)
	} else if !present {
		t.Errorf("Grove indicated that a node was not present when it should have been")
	} else if node == nil {
		t.Errorf("Grove did not return a node when the requested node was present")
	}
}

func TestGroveAdd(t *testing.T) {
	fs := newFakeFS()
	fakeNodeBuilder := NewNodeBuilder(t)
	reply, _ := fakeNodeBuilder.newReplyFile("test content")
	g, err := grove.NewWithFS(fs)
	if err != nil {
		t.Errorf("Failed constructing grove: %v", err)
	}

	if err := g.Add(reply); err != nil {
		t.Errorf("Expected Add() to succeed: %v", err)
	}
}
