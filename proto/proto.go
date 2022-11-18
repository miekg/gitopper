// Package proto holds the structures that return the json to the client.
package proto

type (
	ListMachines struct {
		Machines []string `json:"machines"`
	}

	ListServices struct {
		ListServices []ListService `json:"services"`
	}

	ListService struct {
		Service   string `json:"service"`
		Hash      string `json:"hash"`
		State     string `json:"state"`
		StateInfo string `json:"stateinfo"`
	}
)
