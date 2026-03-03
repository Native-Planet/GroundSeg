package system

import (
	"regexp"
	"strconv"
	"groundseg/system/metrics"
)

// GetMemory returns memory totals using the host metrics service.
func GetMemory() (uint64, uint64, error) {
	return metrics.GetMemory()
}

// GetCPU returns CPU utilization percentage using the host metrics service.
func GetCPU() (int, error) {
	return metrics.GetCPU()
}

// GetDisk returns disk usage statistics using the host metrics service.
func GetDisk() (map[string][2]uint64, error) {
	return metrics.GetDisk()
}

// GetTemp returns CPU temperature using the host metrics service.
func GetTemp() (float64, error) {
	return metrics.GetTemp()
}

func octalToAscii(s string) (string, error) {
	re := regexp.MustCompile(`\\[0-7]{3}`)
	replaceFunc := func(match string) string {
		i, err := strconv.ParseInt(match[1:], 8, 64)
		if err != nil {
			return match
		}
		return string(rune(i))
	}
	decoded := re.ReplaceAllStringFunc(s, replaceFunc)
	return decoded, nil
}
