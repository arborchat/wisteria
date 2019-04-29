package forest_test

import (
	"testing"

	"git.sr.ht/~whereswaldon/forest-go/fields"
)

func QualifiedContentOrSkip(t *testing.T, contentType fields.ContentType, content []byte) *fields.QualifiedContent {
	qContent, err := fields.NewQualifiedContent(contentType, content)
	if err != nil {
		t.Skip("Failed to qualify content", err)
	}
	return qContent
}
