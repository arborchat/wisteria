package main

import (
	"encoding"
	"io/ioutil"
	"strings"

	"git.sr.ht/~whereswaldon/forest-go/fields"
)

// saveAs stores the binary form of the given BinaryMarshaler into a new file called `name`
func saveAs(name string, node encoding.BinaryMarshaler) error {
	b, err := node.MarshalBinary()
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(name, b, 0660); err != nil {
		return err
	}
	return nil
}

// index returns the index of `element` within `group`, or -1 if it is not present
func index(element *fields.QualifiedHash, group []*fields.QualifiedHash) int {
	for i, current := range group {
		if element.Equals(current) {
			return i
		}
	}
	return -1
}

// in returns whether `element` is in `group`
func in(element *fields.QualifiedHash, group []*fields.QualifiedHash) bool {
	return index(element, group) >= 0
}

// nth returns the `n`th rune in the input string. Note that this is not the same as the
// Nth byte of data, as unicode runes can take multiple bytes.
func nth(input string, n int) rune {
	for i, r := range input {
		if i == n {
			return r
		}
	}
	return '?'
}

// stripCommentLines removes all lines in `input` that begin with "#"
func stripCommentLines(input string) string {
	lines := strings.Split(input, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if !strings.HasPrefix(line, "#") {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}
