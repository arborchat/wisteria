package grove_test

import (
	"os"
	"testing"

	"git.sr.ht/~whereswaldon/forest-go/grove"
)

type fakeFS struct {
	files map[string]*os.File
}

var _ grove.FS = fakeFS{}

// Open opens the given path as an absolute path relative to the root
// of the fakeFS
func (r fakeFS) Open(path string) (*os.File, error) {
	return r.files[path], nil
}

// Create makes the given path as an absolute path relative to the root
// of the fakeFS
func (r fakeFS) Create(path string) (*os.File, error) {
	return r.files[path], nil
}

// OpenFile opens the given path as an absolute path relative to the root
// of the fakeFS
func (r fakeFS) OpenFile(path string, flag int, perm os.FileMode) (*os.File, error) {
	return r.files[path], nil
}

func TestCreateEmptyGrove(t *testing.T) {
	fs := fakeFS{
		make(map[string]*os.File),
	}
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
