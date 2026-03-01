package response

import (
	"fmt"
	"regexp"
	"strings"
)

var appStatusRegex = regexp.MustCompile(`app status:\s+([^\s]+)`)

func ParsePokeResponse(resType string, response string) (string, bool, error) {
	responseSlice := strings.Split(response, "\n")

	switch resType {
	case "success":
		for _, line := range responseSlice {
			if strings.Contains(line, "[0 %avow 0 %noun %success]") {
				return "", true, nil
			}
		}
		return "", false, nil
	case "code":
		for _, line := range responseSlice {
			if strings.Contains(line, "%avow") {
				endIndex := strings.Index(line, "]")
				if endIndex == -1 {
					continue
				}
				lastPercentIndex := strings.LastIndex(line[:endIndex], "%")
				if lastPercentIndex != -1 && lastPercentIndex < endIndex {
					code := line[lastPercentIndex+1 : endIndex]
					code = strings.TrimSpace(code)
					return code, false, nil
				}
			}
		}
	case "desk":
		for _, line := range responseSlice {
			if strings.Contains(line, "%avow") {
				if strings.Contains(line, "does not yet exist") {
					return "not-found", false, nil
				}
				match := appStatusRegex.FindStringSubmatch(line)
				if len(match) >= 2 {
					appStatus := strings.TrimSuffix(match[1], "]")
					return appStatus, false, nil
				}
				return "not-found", false, nil
			}
		}
	case "default":
		return "", false, fmt.Errorf("Unknown poke response")
	}
	return "", false, fmt.Errorf("+code not in poke response")
}
