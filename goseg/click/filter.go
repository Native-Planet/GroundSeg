package click

import (
	"fmt"
	"math/big"
	"regexp"
	"strings"

	"github.com/stevelacy/go-urbit/noun"
)

func filterResponse(resType string, response string) (string, bool, error) {
	responseSlice := strings.Split(response, "\n")
	/*
		example usage:
		code, _, err := filterResponse("code",[]string{"some","response"})
		_, ack, err := filterResponse("pack",[]string{"pack","response"})
	*/
	switch resType {
	case "success": // use this if no value need to be returned
		for _, line := range responseSlice {
			if strings.Contains(line, "[0 %avow 0 %noun %success]") {
				return "", true, nil
			}
		}
		return "", false, nil
	case "code":
		for _, line := range responseSlice {
			if strings.Contains(line, "%avow") {
				// Find the last % before the closing ]
				endIndex := strings.Index(line, "]")
				lastPercentIndex := strings.LastIndex(line[:endIndex], "%")

				if lastPercentIndex != -1 && endIndex != -1 && lastPercentIndex < endIndex {
					// Extract the substring
					code := line[lastPercentIndex+1 : endIndex]
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
				// Define a regular expression to match "app status" and capture it
				regex := regexp.MustCompile(`app status:\s+([^\s]+)`)
				// Find the first match in the input string
				match := regex.FindStringSubmatch(line)
				// Check if a match was found
				if len(match) >= 2 {
					appStatus := match[1]
					return appStatus, false, nil
				} else {
					return "not-found", false, nil
				}
				return "not found", false, nil
			}
		}
	case "jam":
		for _, line := range responseSlice {
			if strings.Contains(line, "%avow") {
				var jam string
				// Find the index of "noun "
				index := strings.Index(line, "noun ")

				if index != -1 {
					// Slice the string from just after "noun " onward
					jam = line[index+len("noun "):]
				} else {
					return "", false, fmt.Errorf("Unable to extract jam file from avow")
				}
				jam = strings.TrimPrefix(jam, "0x")
				jamAtom := new(big.Int)
				jamAtom.SetString(jam, 16)
				n := noun.Cue(jamAtom)
				_ = n
				// dump to file
				return "some-file.jam", true, nil
			}
		}
		return "not found", false, fmt.Errorf("Jam file thread failure")
	case "default":
		return "", false, fmt.Errorf("Unknown poke response")
	}
	return "", false, fmt.Errorf("+code not in poke response")
}
