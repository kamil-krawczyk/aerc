package gpgbin

import (
	"bytes"
	"fmt"
	"io"

	"git.sr.ht/~rjarry/aerc/models"
)

// Encrypt runs gpg --encrypt [--sign] -r [recipient]. The default is to have
// --trust-model always set
func Encrypt(r io.Reader, to []string, from string) ([]byte, error) {
	// TODO probably shouldn't have --trust-model always a default
	args := []string{
		"--armor",
		"--trust-model", "always",
	}
	if from != "" {
		args = append(args, "--sign", "--default-key", from)
	}
	for _, rcpt := range to {
		args = append(args, "--recipient", rcpt)
	}
	args = append(args, "--encrypt", "-")

	g := newGpg(r, args)
	err := g.cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("gpg: failed to run encryption: %w", err)
	}
	outRdr := bytes.NewReader(g.stdout.Bytes())
	var md models.MessageDetails
	err = parse(outRdr, &md)
	if err != nil {
		return nil, fmt.Errorf("gpg: failure to encrypt: %v. check public key(s)", err)
	}
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, md.Body)

	return buf.Bytes(), nil
}
