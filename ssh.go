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

func newRouter(c Config, hosts []string) ssh.Handler {
	return func(s ssh.Session) {
		pub := s.PublicKey()
		if pub == nil {
			log.Warningf("Connection denied for user %q", s.User())
			io.WriteString(s, http.StatusText(http.StatusUnauthorized))
			s.Exit(http.StatusUnauthorized)
			return
		}
		ro := false
		for _, a := range c.Keys {
			if ssh.KeysEqual(a.PublicKey, s.PublicKey()) {
				ro = a.RO
			}
		}
		if len(s.Command()) == 0 {
			log.Warningf("No commands in connection for user %q", s.User())
			io.WriteString(s, http.StatusText(http.StatusBadRequest))
			s.Exit(http.StatusBadRequest)
			return
		}
		for prefix, f := range routes {
			if strings.HasPrefix(s.Command()[0], prefix) {
				if ro && strings.HasPrefix(s.Command()[0], "/state/") {
					log.Infof("Key for user %q is set RO and route is RW, denying", s.User())
					io.WriteString(s, http.StatusText(http.StatusUnauthorized))
					s.Exit(http.StatusUnauthorized)
					return
				}
				log.Infof("Routing to %q for user %q", prefix, s.User())
				f(c, s, hosts)
				return
			}
		}

		log.Warningf("No route found for user %q", s.User())
		io.WriteString(s, http.StatusText(http.StatusNotFound))
		s.Exit(http.StatusNotFound)
	}
}

var routes = map[string]func(Config, ssh.Session, []string){
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

func ListMachines(c Config, s ssh.Session, _ []string) {
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

func ListService(c Config, s ssh.Session, hosts []string) {
	ls := proto.ListServices{ListServices: []proto.ListService{}}

	target := ""
	if len(s.Command()) > 1 {
		target = s.Command()[1]
	}
	for _, service := range c.Services {
		if !service.forMe(hosts) {
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

func FreezeService(c Config, s ssh.Session, hosts []string) {
	freezeStateService(c, s, StateFreeze, hosts)
}

func UnfreezeService(c Config, s ssh.Session, hosts []string) {
	freezeStateService(c, s, StateOK, hosts)
}

func myServices(c Config, target string, hosts []string) []*Service {
	var s []*Service
	for _, serv := range c.Services {
		if serv.forMe(hosts) && serv.Service == target {
			s = append(s, serv)
		}
	}
	return s
}

func freezeStateService(c Config, s ssh.Session, state State, hosts []string) {
	if len(s.Command()) < 2 {
		s.Exit(http.StatusNotAcceptable)
		return
	}
	target := s.Command()[1]
	for _, serv := range myServices(c, target, hosts) {
		serv.SetState(state, "")
		log.Infof("Machine %q, service %q set to %s", serv.Machine, serv.Service, state)
		io.WriteString(s, http.StatusText(http.StatusOK))
		s.Exit(0)
		return
	}
	io.WriteString(s, http.StatusText(http.StatusNotFound))
	s.Exit(http.StatusNotFound)
}

func RollbackService(c Config, s ssh.Session, hosts []string) {
	if len(s.Command()) < 3 {
		return
	}
	target := s.Command()[1]
	hash := s.Command()[2]
	if _, err := hex.DecodeString(hash); err != nil {
		io.WriteString(s, http.StatusText(http.StatusNotAcceptable)+", not a valid hexadecimal git hash: "+hash)
		s.Exit(http.StatusNotAcceptable)
		return
	}

	for _, serv := range myServices(c, target, hosts) {
		serv.SetState(StateRollback, hash)
		log.Infof("Machine %q, service %q set to %s", serv.Machine, serv.Service, StateRollback)
		io.WriteString(s, http.StatusText(http.StatusOK))
		s.Exit(0)
		return
	}
	io.WriteString(s, http.StatusText(http.StatusNotFound))
	s.Exit(http.StatusNotFound)
}
