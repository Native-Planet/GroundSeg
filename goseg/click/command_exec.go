package click

import "fmt"

func executeClickCommand(patp, file, hoon, sourcePath, successToken, operation string) (string, error) {
	if err := createHoon(patp, file, hoon); err != nil {
		return "", fmt.Errorf("%s failed to create hoon: %v", operation, err)
	}
	defer deleteHoon(patp, file)

	response, err := clickExec(patp, file, sourcePath)
	if err != nil {
		return "", fmt.Errorf("%s failed to execute hoon: %v", operation, err)
	}
	if successToken != "" {
		_, success, err := filterResponse(successToken, response)
		if err != nil {
			return "", fmt.Errorf("%s failed to parse response: %v", operation, err)
		}
		if !success {
			return "", fmt.Errorf("%s failed poke", operation)
		}
	}
	return response, nil
}
