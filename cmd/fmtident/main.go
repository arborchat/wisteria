package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"

	forest "git.sr.ht/~whereswaldon/forest-go"
)

func prettyPrintFrom(input io.Reader) {
	var in bytes.Buffer
	if _, err := in.ReadFrom(input); err != nil && err != io.EOF {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	i, err := forest.UnmarshalIdentity(in.Bytes())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	text, err := json.Marshal(i)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if _, err := os.Stdout.Write(text); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func main() {
	inputs := make([]io.Reader, 0, 10)
	if len(os.Args) < 2 {
		inputs = append(inputs, os.Stdin)
	} else {
		for _, name := range os.Args[1:] {
			var (
				file io.Reader
				err  error
			)
			if name == "-" {
				file = os.Stdin
			} else {
				file, err = os.Open(name)
			}
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			inputs = append(inputs, file)
		}
	}
	for _, file := range inputs {
		prettyPrintFrom(file)
	}
}
