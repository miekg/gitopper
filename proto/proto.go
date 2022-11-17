// Package proto holds the structures that return the json to the client.
package proto

type (
	ListMachine struct {
		Machines []string
	}

	ListServices struct {
		Services []string
	}

	ListService struct {
		Service string
		Hash    string
		State   string
	}
)
