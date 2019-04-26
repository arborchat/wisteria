package forest_test

import (
	"testing"

	forest "git.sr.ht/~whereswaldon/forest-go"
)

func QualifiedContentOrSkip(t *testing.T, contentType forest.ContentType, content []byte) *forest.QualifiedContent {
	qContent, err := forest.NewQualifiedContent(contentType, content)
	if err != nil {
		t.Skip("Failed to qualify content", err)
	}
	return qContent
}
