package main

import (
	"bytes"
	"fmt"
	"io"
	"os"

	forest "git.sr.ht/~whereswaldon/forest-go"
)

func main() {
	var in bytes.Buffer
	if _, err := in.ReadFrom(os.Stdin); err != nil && err != io.EOF {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	i, err := forest.UnmarshalIdentity(in.Bytes())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("%v\n", i)
}
