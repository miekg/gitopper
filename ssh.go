package main

import (
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/gliderlabs/ssh"
	"go.science.ru.nl/log"
)

var sshRoutes = map[string]func(Config, ssh.Session){
	"/state/freeze/": RollbackService,
}

func newSSHRouter(c Config) {
	// generate persistent key, or ignore client side.
	ssh.Handle(func(s ssh.Session) {
		if len(s.Command()) == 0 {
			// exit code
			return
		}
		for prefix, f := range sshRoutes {
			if strings.HasPrefix(s.Command()[0], prefix) {
				f(c, s)
				return
			}
		}

		// error
		io.WriteString(s, fmt.Sprintf("Hello %s %v\n", s.User(), s.Command()))
	})

	// parse pub keys in start up into Config

	log.Info("Starting ssh server on port 2222...")
	ssh.ListenAndServe(":2222", nil,
		ssh.PublicKeyAuth(func(ctx ssh.Context, key ssh.PublicKey) bool {
			data, _ := ioutil.ReadFile("/home/miek/.ssh/id_ed25519.pub")
			allowed, _, _, _, _ := ssh.ParseAuthorizedKey(data)
			return ssh.KeysEqual(key, allowed)
		}),
	)
}

func RollbackService(c Config, s ssh.Session) {
	if len(s.Command()) < 3 {
		return
	}
	service := s.Command()[1]
	hash := s.Command()[2]
	if _, err := hex.DecodeString(hash); err != nil {
		//		http.Error(w, http.StatusText(http.StatusNotAcceptable)+", not a valid git hash: "+vars["hash"], http.StatusNotFound)
		return
	}

	for _, s := range c.Services {
		if s.Service == service {
			s.SetState(StateRollback, hash)
			log.Infof("Machine %q, service %q set to %s", s.Machine, s.Service, StateRollback)
			// http.Error(w, http.StatusText(http.StatusOK), http.StatusOK)
			return
		}
	}
	// http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
}
