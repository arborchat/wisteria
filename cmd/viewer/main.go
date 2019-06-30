package main

import (
	"flag"
	"log"

	forest "git.sr.ht/~whereswaldon/forest-go"
)

func main() {
	var (
		gpguser string
	)
	flag.StringVar(&gpguser, "gpguser", "", "[required] the name of the gpg identity to use")
	flag.Parse()
	if gpguser == "" {
		log.Fatal("--gpguser is required")
	}
	signer, err := forest.NewGPGSigner(gpguser)
	if err != nil {
		log.Fatal(err)
	}
	id, err := forest.NewIdentity(signer, "viewer", "")
	if err != nil {
		log.Fatal(err)
	}

	comm, err := forest.As(id, signer).NewCommunity("viewer", "")
	if err != nil {
		log.Fatal(err)
	}

	reply, err := forest.As(id, signer).NewReply(comm, "Hello, World!", "")
	if err != nil {
		log.Fatal(err)
	}

	store := forest.NewMemoryStore()
	for _, node := range []forest.Node{id, comm, reply} {
		if err := store.Add(node); err != nil {
			log.Fatal(err)
		}
	}

	if err := render(store); err != nil {
		log.Fatal(err)
	}
}

func render(store forest.Store) error {
	return nil
}
