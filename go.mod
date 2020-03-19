module git.sr.ht/~whereswaldon/wisteria

require (
	git.sr.ht/~whereswaldon/forest-go v0.0.0-20200319194448-e3a47dd95cda
	git.sr.ht/~whereswaldon/sprout-go v0.0.0-20200319194723-df82b3bc1ee9
	github.com/0xAX/notificator v0.0.0-20181105090803-d81462e38c21
	github.com/awnumar/memguard v0.21.0
	github.com/bbrks/wrap/v2 v2.3.1-0.20191113183707-81f8a5d714b8
	github.com/gdamore/tcell v1.3.0
	github.com/mattn/go-runewidth v0.0.4
	github.com/pkg/profile v1.3.0
	golang.org/x/crypto v0.0.0-20191122220453-ac88ee75c92c
	golang.org/x/sys v0.0.0-20191224085550-c709ea063b76 // indirect
	golang.org/x/text v0.3.2 // indirect
)

replace golang.org/x/crypto => github.com/ProtonMail/crypto v0.0.0-20191122234321-e77a1f03baa0

go 1.13
