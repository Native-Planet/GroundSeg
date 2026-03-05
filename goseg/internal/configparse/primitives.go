package configparse

import "fmt"

func Bool(name string, value any) (bool, error) {
	parsed, ok := value.(bool)
	if !ok {
		return false, fmt.Errorf("invalid %s value: %T", name, value)
	}
	return parsed, nil
}

func Int(name string, value any) (int, error) {
	switch parsed := value.(type) {
	case int:
		return parsed, nil
	case float64:
		return int(parsed), nil
	default:
		return 0, fmt.Errorf("invalid %s value: %T", name, value)
	}
}

func String(name string, value any) (string, error) {
	parsed, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("invalid %s value: %T", name, value)
	}
	return parsed, nil
}
