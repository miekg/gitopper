package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/miekg/gitopper/proto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.science.ru.nl/log"
)

func newRouter(c Config) *mux.Router {
	router := mux.NewRouter()
	router.Path("/metrics").Handler(promhttp.Handler())

	// listing
	router.Path("/list/machines").Methods("GET").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ListMachines(c, w, r)
	})
	// don't really need a seperate one for this, can be /service without a service
	router.Path("/list/services").Methods("GET").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ListServices(c, w, r)
	})
	router.Path("/list/service/{service}").Methods("GET").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ListService(c, w, r)
	})

	// state changes
	router.Path("/state/freeze/{service}").Methods("POST").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		FreezeService(c, StateFreeze, w, r)
	})
	router.Path("/state/unfreeze/{service}").Methods("POST").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		FreezeService(c, StateOK, w, r)
	})
	router.Path("/state/rollback/{service}/{hash}").Methods("POST").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		RollbackService(c, w, r)
	})
	return router
}

func ListMachines(c Config, w http.ResponseWriter, r *http.Request) {
	lm := proto.ListMachines{
		Machines: make([]string, len(c.Services)),
	}
	for i, service := range c.Services {
		lm.Machines[i] = service.Machine
	}
	data, err := json.Marshal(lm)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func ListServices(c Config, w http.ResponseWriter, r *http.Request) {
	ls := proto.ListServices{
		ListServices: make([]proto.ListService, len(c.Services)),
	}
	for i, service := range c.Services {
		state, info := service.State()
		ls.ListServices[i] = proto.ListService{
			Service:     service.Service,
			Hash:        service.Hash(),
			State:       state.String(),
			StateInfo:   info,
			StateChange: service.Change().Format(time.RFC1123),
		}
	}
	data, err := json.Marshal(ls)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func ListService(c Config, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	for _, service := range c.Services {
		if service.Service == vars["service"] {
			state, info := service.State()
			ls := proto.ListService{
				Service:     service.Service,
				Hash:        service.Hash(),
				State:       state.String(),
				StateInfo:   info,
				StateChange: service.Change().String(),
			}
			data, err := json.Marshal(ls)
			if err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(data)
			return
		}
	}
	http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
}

func FreezeService(c Config, state State, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	for _, service := range c.Services {
		if service.Service == vars["service"] {
			service.SetState(state, "")
			log.Infof("Machine %q, service %q set to %s", service.Machine, service.Service, state)
			http.Error(w, http.StatusText(http.StatusOK), http.StatusOK)
			return
		}
	}
	http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
}

func RollbackService(c Config, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	for _, service := range c.Services {
		if service.Service == vars["service"] {
			service.SetState(StateRollback, vars["hash"])
			log.Infof("Machine %q, service %q set to %s", service.Machine, service.Service, StateRollback)
			http.Error(w, http.StatusText(http.StatusOK), http.StatusOK)
			return
		}
	}
	http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
}
