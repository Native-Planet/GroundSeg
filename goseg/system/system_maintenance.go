package system

import "groundseg/system/maintenance"

func IsNPBox(basePath string) bool {
	return maintenance.IsNPBox(basePath)
}

func FixerScript(basePath string) error {
	return maintenance.FixerScript(basePath)
}
