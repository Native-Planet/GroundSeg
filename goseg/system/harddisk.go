package system

import (
	"encoding/json"
	"fmt"
	"groundseg/structs"
)

func ListHardDisks() (structs.LSBLKDevice, error) {
	var dev structs.LSBLKDevice
	out, err := runCommand("lsblk", "--json", "--bytes")
	if err != nil {
		return dev, fmt.Errorf("Failed to lsblk: %v", err)
	}
	err = json.Unmarshal([]byte(out), &dev)
	if err != nil {
		return dev, fmt.Errorf("Failed to unmarshal lsblk json: %v", err)
	}
	return dev, nil
}
