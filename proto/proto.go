// Package proto holds the structures that return the json to the client.
package proto

type (
	ListMachines struct {
		ListMachines []ListMachine `json:"machines"`
	}

	ListMachine struct {
		Machine string `json:"machine"` // Machine as set in config file.
		Actual  string `json:"actual"`  // Actual machine responding (i.e. -h flag might be used)
	}

	ListServices struct {
		ListServices []ListService `json:"services"`
	}

	ListService struct {
		Service     string `json:"service"`
		Hash        string `json:"hash"`
		State       string `json:"state"`
		StateInfo   string `json:"stateinfo"`
		StateChange string `json:"change"`
	}

	ListConfigs struct {
		ListConfigs []ListConfig `json:"config"`
	}

	ListConfig struct {
		Service  string   `json:"service"`
		Upstream string   `json:"upstream"`
		Systemd  string   `json:"systemd"`
		Dirs     []string `json:"dir"`
		Locals   []string `json:"local"`
	}
)
