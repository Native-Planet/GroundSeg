package structs

type LSBLKDevice struct {
	BlockDevices []BlockDev `json:"blockdevices"`
}

type BlockDev struct {
	UUID        string     `json:"uuid"`
	Name        string     `json:"name"`
	MajMin      string     `json:"maj:min"`
	FSType      string     `json:"fstype"`
	RM          bool       `json:"rm"`
	Size        int        `json:"size"`
	Ro          bool       `json:"ro"`
	Type        string     `json:"type"`
	Mountpoints []string   `json:"mountpoints"`
	Children    []BlockDev `json:"children"`
}
