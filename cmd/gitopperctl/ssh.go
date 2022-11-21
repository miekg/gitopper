package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os/user"
	"strings"

	"github.com/urfave/cli/v2"
	"golang.org/x/crypto/ssh"
)

func querySSH(ctx *cli.Context, at, command string, args ...string) ([]byte, error) {
	ident := ctx.String("i")
	if ident == "" {
		return nil, fmt.Errorf("identity file not given, -i flag")
	}
	port := ctx.String("p")
	if port == "" {
		port = "2222"
	}
	at = at + ":" + port

	key, err := ioutil.ReadFile(ident)
	if err != nil {
		return nil, err
	}

	// Create the Signer for this private key.
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, err
	}

	user, err := user.Current()
	if err != nil {
		return nil, err
	}

	config := &ssh.ClientConfig{
		User:            user.Username,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
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

	// makes this buffer bounded...?
	stdoutBuf := &bytes.Buffer{}
	ss.Stdout = stdoutBuf

	cmdline := command + " " + strings.Join(args, " ")
	if err := ss.Run(cmdline); err != nil {
		return nil, err
	}
	return stdoutBuf.Bytes(), nil
}
