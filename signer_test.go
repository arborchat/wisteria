package forest_test

import (
	"encoding/base64"
	"io/ioutil"
	"os"
	"os/exec"
	"testing"

	forest "git.sr.ht/~whereswaldon/forest-go"
)

// ensureGPGInstalled will cause the calling test to be skipped if GPG
// isn't available on the system.
func ensureGPGInstalled(t *testing.T) {
	gpg2 := exec.Command("gpg2", "--version")
	if err := gpg2.Run(); err != nil {
		t.Skip("GPG2 not available", err)
		t.SkipNow()
	}
}

const testPassphrase = "arborchat-testing-key"
const testUsername = testPassphrase
const testData = testPassphrase

// TestGPGSigner creates a new GPG key in a temporary directory and signs some data.
func TestGPGSigner(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping expensive GPG test in short mode")
	}
	signer, cleanup := getGPGSignerOrFail(t)
	defer cleanup()
	// sign some data
	signature, err := signer.Sign([]byte(testData))
	if err != nil {
		t.Errorf("Failed sign data: %v", err)
	}
	t.Logf("data: %s\nsignature %s", testData, base64.StdEncoding.EncodeToString(signature))
}

func getGPGSignerOrFail(t *testing.T) (forest.Signer, func()) {
	ensureGPGInstalled(t)
	// generate PGP key to use
	tempdir, err := ioutil.TempDir("", "arborchat-test")
	if err != nil {
		t.Errorf("Failed to create temporary GNUPG home: %v", err)
	}
	cleanup := func() { os.RemoveAll(tempdir) }
	gpg2 := exec.Command("gpg2", "--yes", "--batch", "--pinentry-mode", "loopback", "--passphrase", testPassphrase, "--quick-generate-key", testUsername)
	gpg2.Env = []string{"GNUPGHOME=" + tempdir}
	stderr, _ := gpg2.StderrPipe()
	if err := gpg2.Run(); err != nil {
		data, _ := ioutil.ReadAll(stderr)
		t.Log(data)
		t.Errorf("Error generating key: %v", err)
		cleanup()
	}
	// build signer
	signer, err := forest.NewGPGSigner(testUsername)
	if err != nil {
		t.Errorf("Failed to construct signer with valid username: %v", err)
		cleanup()
	}
	signer.Rewriter = func(gpg2 *exec.Cmd) error {
		gpg2.Args = append(append(gpg2.Args[:1], "--yes", "--batch", "--pinentry-mode", "loopback", "--passphrase", testPassphrase), gpg2.Args[1:]...)
		gpg2.Env = []string{"GNUPGHOME=" + tempdir}
		gpg2.Stderr = os.Stderr
		return nil
	}
	return signer, cleanup
}

func TestGPGSignerAsIdentity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping expensive GPG test in short mode")
	}
	signer, cleanup := getGPGSignerOrFail(t)
	defer cleanup()
	identity, err := forest.NewIdentity(signer, "test name", "")
	if err != nil {
		t.Fatal("Failed to create Identity with valid parameters", err)
	}
	if correct, err := forest.ValidateID(identity, *identity.ID()); err != nil || !correct {
		t.Error("ID validation failed on unmodified node", err)
	}
	if correct, err := forest.ValidateSignature(identity, identity); err != nil || !correct {
		t.Error("Signature validation failed on unmodified node", err)
	}
}
