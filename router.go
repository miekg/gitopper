package main

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/miekg/gitopper/proto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func newRouter(c Config) *mux.Router {
	router := mux.NewRouter()
	router.Path("/metrics").Handler(promhttp.Handler())
	router.Path("/list/machines").Methods("GET").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ListMachines(c, w, r)
	})
	router.Path("/list/services").Methods("GET").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ListServices(c, w, r)
	})
	router.Path("/list/service/{service}").Methods("GET").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ListService(c, w, r)
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
		Services: make([]string, len(c.Services)),
	}
	for i, service := range c.Services {
		ls.Services[i] = service.Service
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
			gc := service.newGitCmd() // TODO: also in state, as to minimize forks.
			ls := proto.ListService{
				Service: service.Service,
				Hash:    gc.Hash(),
				State:   service.State().String(),
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
