package main

import (
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/gliderlabs/ssh"
	"github.com/miekg/gitopper/osutil"
	"github.com/miekg/gitopper/proto"
	"go.science.ru.nl/log"
)

var sshRoutes = map[string]func(Config, ssh.Session){
	"/list/machines": ListMachines,
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

		io.WriteString(s, http.StatusText(http.StatusNotFound))
		s.Exit(http.StatusNotFound)
	})

	// parse pub keys in start up into Config
}

func ListMachines(c Config, s ssh.Session) {
	lm := proto.ListMachines{
		ListMachines: make([]proto.ListMachine, len(c.Services)),
	}
	for i, service := range c.Services {
		lm.ListMachines[i] = proto.ListMachine{
			Machine: service.Machine,
			Actual:  osutil.Hostname(),
		}
	}
	data, err := json.Marshal(lm)
	if err != nil {
		io.WriteString(s, http.StatusText(http.StatusInternalServerError))
		s.Exit(http.StatusInternalServerError)
		return
	}
	s.Write(data)
}

func RollbackService(c Config, s ssh.Session) {
	if len(s.Command()) < 3 {
		return
	}
	service := s.Command()[1]
	hash := s.Command()[2]
	if _, err := hex.DecodeString(hash); err != nil {
		io.WriteString(s, http.StatusText(http.StatusNotAcceptable)+", not a valid git hash: "+hash)
		s.Exit(http.StatusNotFound)
		return
	}

	for _, serv := range c.Services {
		if serv.Service == service {
			serv.SetState(StateRollback, hash)
			log.Infof("Machine %q, service %q set to %s", serv.Machine, serv.Service, StateRollback)
			io.WriteString(s, http.StatusText(http.StatusOK))
			s.Exit(0)
			return
		}
	}
	io.WriteString(s, http.StatusText(http.StatusNotFound))
	s.Exit(http.StatusNotFound)
}
