package structs

// broadcast payload object struct
type AuthBroadcast struct {
	Type      string           `json:"type"`
	AuthLevel string           `json:"auth_level"`
	Upload    Upload           `json:"upload"`
	Logs      Logs             `json:"logs"`
	NewShip   NewShip          `json:"newShip"`
	System    System           `json:"system"`
	Profile   Profile          `json:"profile"`
	Urbits    map[string]Urbit `json:"urbits"`
}

// new ship
type NewShip struct {
	Transition struct {
		BootStage string `json:"bootStage"`
		Patp      string `json:"patp"`
		Error     string `json:"error"`
	} `json:"transition"`
}

// broadcast payload subobject
type System struct {
	Info struct {
		Usage   SystemUsage   `json:"usage"`
		Updates SystemUpdates `json:"updates"`
		Wifi    SystemWifi    `json:"wifi"`
	} `json:"info"`
	Transition SystemTransitionBroadcast `json:"transition"`
}

type SystemTransitionBroadcast struct {
	Swap int `json:"swap"`
	Type string `json:"type"`
}

// broadcast payload subobject
type SystemUsage struct {
	RAM      []uint64 `json:"ram"`
	CPU      int      `json:"cpu"`
	CPUTemp  float64  `json:"cpu_temp"`
	Disk     []uint64 `json:"disk"`
	SwapFile int      `json:"swap"`
}

// broadcast payload subobject
type SystemUpdates struct {
	Linux struct {
		State   string `json:"state"`
		Upgrade int    `json:"upgrade"`
		New     int    `json:"new"`
		Remove  int    `json:"remove"`
		Ignore  int    `json:"ignore"`
	} `json:"linux"`
}

// broadcast payload subobject
type SystemWifi struct {
	Status   string   `json:"status"`
	Active   string   `json:"active"`
	Networks []string `json:"networks"`
}

// broadcast payload subobject
type Profile struct {
	Startram Startram `json:"startram"`
}

// broadcast payload subobject
type Startram struct {
	Info struct {
		Registered bool                      `json:"registered"`
		Running    bool                      `json:"running"`
		Region     any                       `json:"region"`
		Expiry     any                       `json:"expiry"`
		Renew      bool                      `json:"renew"`
		Endpoint   string                    `json:"endpoint"`
		Regions    map[string]StartramRegion `json:"regions"`
	} `json:"info"`
	Transition struct {
		Endpoint string `json:"endpoint"`
		Register any    `json:"register"`
		Toggle   any    `json:"toggle"`
	} `json:"transition"`
}

// broadcast payload subobject
type Urbit struct {
	Info struct {
		Network          string `json:"network"`
		Running          bool   `json:"running"`
		URL              string `json:"url"`
		UrbAlias         bool   `json:"urbAlias"`
		MemUsage         uint64 `json:"memUsage"`
		DiskUsage        int64  `json:"diskUsage"`
		LoomSize         int    `json:"loomSize"`
		DevMode          bool   `json:"devMode"`
		DetectBootStatus bool   `json:"detectBootStatus"`
		Remote           bool   `json:"remote"`
		Vere             any    `json:"vere"`
	} `json:"info"`
	Transition UrbitTransitionBroadcast `json:"transition"`
}

// broadcast payload subobject
type UrbitTransitionBroadcast struct {
	Meld                      string `json:"meld"`
	ServiceRegistrationStatus string `json:"serviceRegistrationStatus"`
	TogglePower               string `json:"togglePower"`
	DeleteShip                string `json:"deleteShip"`
}

// used to construct broadcast pier info subobject
type ContainerStats struct {
	MemoryUsage uint64
	DiskUsage   int64
}

// broadcast payload subobject
type Logs struct {
	Containers struct {
		Wireguard struct {
			Logs []any `json:"logs"`
		} `json:"wireguard"`
	} `json:"containers"`
	System struct {
		Stream bool  `json:"stream"`
		Logs   []any `json:"logs"`
	} `json:"system"`
}

// broadcast payload subobject
type Upload struct {
	Status   string `json:"status"`
	Size     int    `json:"size"`
	Uploaded int    `json:"uploaded"`
	Patp     any    `json:"patp"`
}

// broadcast payload subobject
type UnauthBroadcast struct {
	Type      string `json:"type"`
	AuthLevel string `json:"auth_level"`
	Login     struct {
		Remainder int `json:"remainder"`
	} `json:"login"`
}

// broadcast payload subobject
type SetupBroadcast struct {
	Type      string                    `json:"type"`
	AuthLevel string                    `json:"auth_level"`
	Stage     string                    `json:"stage"`
	Page      string                    `json:"page"`
	Regions   map[string]StartramRegion `json:"regions"`
}

// broadcast subobject
type LoginStatus struct {
	Locked   bool
	End      string
	Attempts int
}

// broadcast subobject
type LoginKeys struct {
	Old struct {
		Pub  string
		Priv string
	}
	Cur struct {
		Pub  string
		Priv string
	}
}
