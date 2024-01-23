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
	Apps      Apps             `json:"apps"`
}

// third party integrations
type Apps struct {
	Penpai PenpaiBroadcast `json:"penpai"`
}

type PenpaiBroadcast struct {
	Info struct {
		Allowed     bool     `json:"allowed"`
		Running     bool     `json:"running"`
		ActiveCores int      `json:"activeCores"`
		MaxCores    int      `json:"maxCores"`
		Models      []string `json:"models"`
		ActiveModel string   `json:"activeModel"`
	} `json:"info"`
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
	Swap           bool     `json:"swap"`
	Type           string   `json:"type"`
	Error          []string `json:"error"`
	BugReport      string   `json:"bugReport"`
	BugReportError string   `json:"bugReportError"`
	WifiConnect    string   `json:"wifiConnect"`
}

// broadcast payload subobject
type SystemUsage struct {
	RAM      []uint64             `json:"ram"`
	CPU      int                  `json:"cpu"`
	CPUTemp  float64              `json:"cpu_temp"`
	Disk     map[string][2]uint64 `json:"disk"`
	SwapFile int                  `json:"swap"`
}

// broadcast payload subobject
type SystemUpdates struct {
	Linux struct {
		Upgrade int `json:"upgrade"`
		New     int `json:"new"`
		Remove  int `json:"remove"`
		Ignore  int `json:"ignore"`
	} `json:"linux"`
}

// broadcast payload subobject
type SystemWifi struct {
	Status   bool     `json:"status"`
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
		UrlID      string                    `json:"urlID"`
		Renew      bool                      `json:"renew"`
		Endpoint   string                    `json:"endpoint"`
		Regions    map[string]StartramRegion `json:"regions"`
	} `json:"info"`
	Transition StartramTransition `json:"transition"`
}

type StartramTransition struct {
	Endpoint string `json:"endpoint"`
	Register any    `json:"register"`
	Toggle   any    `json:"toggle"`
	Restart  string `json:"restart"`
}

// broadcast payload subobject
type Urbit struct {
	Info struct {
		LusCode            string `json:"lusCode"`
		Network            string `json:"network"`
		Running            bool   `json:"running"`
		URL                string `json:"url"`
		UrbitAlias         string `json:"urbitAlias"`
		MinIOAlias         string `json:"minioAlias"`
		ShowUrbAlias       bool   `json:"showUrbAlias"`
		MemUsage           uint64 `json:"memUsage"`
		DiskUsage          int64  `json:"diskUsage"`
		LoomSize           int    `json:"loomSize"`
		DevMode            bool   `json:"devMode"`
		DetectBootStatus   bool   `json:"detectBootStatus"`
		Remote             bool   `json:"remote"`
		RemoteReady        bool   `json:"remoteReady"` // remote reg is ready
		Vere               any    `json:"vere"`
		MinIOUrl           string `json:"minioUrl"`
		MinIOPwd           string `json:"minioPwd"`
		LastPack           string `json:"lastPack"`
		NextPack           string `json:"nextPack"`
		PackScheduleActive bool   `json:"packScheduleActive"`
		PackIntervalType   string `json:"packIntervalType"`
		PackIntervalValue  int    `json:"packIntervalValue"`
		PackTime           string `json:"packTime"`
		PackDay            string `json:"packDay"`
		PackDate           int    `json:"packDate"`
		PenpaiCompanion    bool   `json:"penpaiCompanion"`
		Gallseg            bool   `json:"gallseg"`
		MinIOLinked        bool   `json:"minioLinked"`
		StartramReminder   bool   `json:"startramReminder"`
	} `json:"info"`
	Transition UrbitTransitionBroadcast `json:"transition"`
}

// broadcast payload subobject
type UrbitTransitionBroadcast struct {
	Pack                      string `json:"pack"`
	PackMeld                  string `json:"packMeld"`
	ServiceRegistrationStatus string `json:"serviceRegistrationStatus"`
	TogglePower               string `json:"togglePower"`
	ToggleNetwork             string `json:"toggleNetwork"`
	ToggleDevMode             string `json:"toggleDevMode"`
	ToggleMinIOLink           string `json:"toggleMinIOLink"`
	DeleteShip                string `json:"deleteShip"`
	ExportShip                string `json:"exportShip"`
	ShipCompressed            int    `json:"shipCompressed"`
	ExportBucket              string `json:"exportBucket"`
	BucketCompressed          int    `json:"bucketCompressed"`
	RebuildContainer          string `json:"rebuildContainer"`
	Loom                      string `json:"loom"`
	UrbitDomain               string `json:"urbitDomain"`
	MinIODomain               string `json:"minioDomain"`
	PenpaiCompanion           string `json:"penpaiCompanion"`
	Gallseg                   string `json:"gallseg"`
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
	Status    string `json:"status"`
	Patp      string `json:"patp"`
	Error     string `json:"error"`
	Extracted int64  `json:"extracted"`
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
type C2CBroadcast struct {
	Type  string   `json:"type"`
	SSIDS []string `json:"ssids"`
}

// broadcast payload subobject
type SetupBroadcast struct {
	Type      string                    `json:"type"`
	AuthLevel string                    `json:"auth_level"`
	Stage     string                    `json:"stage"`
	Page      int                       `json:"page"`
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
