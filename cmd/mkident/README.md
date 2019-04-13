# mkident

Create Identity nodes in the Arbor Forest.

**WARNING**: While `mkident` does use OpenPGP keys, it is not currently considered secure. It does not use passphrases for the OpenPGP private keys that it manages, meaning that they can be trivially read from disk. This weakness will be addressed in the future. Use at your own risk.

## Build

`go build`

## Usage

See `mkident -h`.
