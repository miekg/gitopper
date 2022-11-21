package main

import (
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gliderlabs/ssh"
	"github.com/miekg/gitopper/osutil"
	"github.com/miekg/gitopper/proto"
	"go.science.ru.nl/log"
)

func newRouter(c Config) {
	ssh.Handle(func(s ssh.Session) {
		if len(s.Command()) == 0 {
			io.WriteString(s, http.StatusText(http.StatusBadRequest))
			s.Exit(http.StatusBadRequest)
			return
		}
		for prefix, f := range routes {
			if strings.HasPrefix(s.Command()[0], prefix) {
				f(c, s)
				return
			}
		}

		io.WriteString(s, http.StatusText(http.StatusNotFound))
		s.Exit(http.StatusNotFound)
	})
}

var routes = map[string]func(Config, ssh.Session){
	"/list/machine":   ListMachines,
	"/list/service":   ListService,
	"/state/freeze":   FreezeService,
	"/state/unfreeze": UnfreezeService,
	"/state/rollback": RollbackService,
}

func writeAndExit(s ssh.Session, data []byte, err error) {
	if err != nil {
		io.WriteString(s, http.StatusText(http.StatusInternalServerError))
		s.Exit(http.StatusInternalServerError)
		return
	}
	s.Write(data)
	s.Exit(0)
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
	writeAndExit(s, data, err)
}

func ListService(c Config, s ssh.Session) {
	ls := proto.ListServices{ListServices: []proto.ListService{}}

	target := ""
	if len(s.Command()) > 1 {
		target = s.Command()[1]
	}
	for _, service := range c.Services {
		if !service.forMe(flagHosts) {
			continue
		}
		state, info := service.State()
		switch {
		case target == "":
			ls.ListServices = append(ls.ListServices, proto.ListService{
				Service:     service.Service,
				Hash:        service.Hash(),
				State:       state.String(),
				StateInfo:   info,
				StateChange: service.Change().Format(time.RFC1123),
			})
		case target != "":
			if service.Service == target {
				ls.ListServices = append(ls.ListServices, proto.ListService{
					Service:     service.Service,
					Hash:        service.Hash(),
					State:       state.String(),
					StateInfo:   info,
					StateChange: service.Change().Format(time.RFC1123),
				})
				break
			}
		}
	}
	if len(ls.ListServices) == 0 {
		io.WriteString(s, http.StatusText(http.StatusNotFound))
		s.Exit(http.StatusNotFound)
		return
	}
	data, err := json.Marshal(ls)
	writeAndExit(s, data, err)
}

func FreezeService(c Config, s ssh.Session) { freezeStateService(c, s, StateFreeze) }

func UnfreezeService(c Config, s ssh.Session) { freezeStateService(c, s, StateOK) }

func freezeStateService(c Config, s ssh.Session, state State) {
	if len(s.Command()) < 2 {
		s.Exit(http.StatusNotAcceptable)
		return
	}
	target := s.Command()[1]
	for _, service := range c.Services {
		if !service.forMe(flagHosts) {
			continue
		}
		if service.Service == target {
			service.SetState(state, "")
			log.Infof("Machine %q, service %q set to %s", service.Machine, service.Service, state)
			io.WriteString(s, http.StatusText(http.StatusOK))
			s.Exit(0)
			return
		}
	}
	io.WriteString(s, http.StatusText(http.StatusNotFound))
	s.Exit(http.StatusNotFound)
}

func RollbackService(c Config, s ssh.Session) {
	if len(s.Command()) < 3 {
		return
	}
	target := s.Command()[1]
	hash := s.Command()[2]
	if _, err := hex.DecodeString(hash); err != nil {
		io.WriteString(s, http.StatusText(http.StatusNotAcceptable)+", not a valid git hash: "+hash)
		s.Exit(http.StatusNotAcceptable)
		return
	}

	for _, service := range c.Services {
		if !service.forMe(flagHosts) {
			continue
		}
		if service.Service == target {
			service.SetState(StateRollback, hash)
			log.Infof("Machine %q, service %q set to %s", service.Machine, service.Service, StateRollback)
			io.WriteString(s, http.StatusText(http.StatusOK))
			s.Exit(0)
			return
		}
	}
	io.WriteString(s, http.StatusText(http.StatusNotFound))
	s.Exit(http.StatusNotFound)
}
