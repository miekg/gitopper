package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"strings"

	"golang.org/x/crypto/ssh"
)

func querySSH(at, command string, args ...string) ([]byte, error) {
	at = at + ":2222"
	key, err := ioutil.ReadFile("/local/home/miek/.ssh/id_ed25519_gitopper")
	if err != nil {
		return nil, err
	}

	// Create the Signer for this private key.
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		log.Fatalf("unable to parse private key: %v", err)
	}

	config := &ssh.ClientConfig{
		User: "miek",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	client, err := ssh.Dial("tcp", at, config)
	if err != nil {
		return nil, err
	}
	defer client.Close()
	ss, err := client.NewSession()
	if err != nil {
		return nil, err
	}
	defer ss.Close()

	stdoutBuf := &bytes.Buffer{}
	ss.Stdout = stdoutBuf

	cmdline := command + " " + strings.Join(args, " ")
	if err := ss.Run(cmdline); err != nil {
		return nil, err
	}
	return stdoutBuf.Bytes(), nil
}
