package docker

func IsMaintenanceBootStatus(status string) bool {
	switch status {
	case "pack", "meld", "chop", "rollchop", "prep", "roll":
		return true
	default:
		return false
	}
}

func PersistentBootStatusAfterContainerBuild(status string) string {
	switch status {
	case "pack", "meld", "chop", "rollchop", "noboot":
		return "noboot"
	case "ignore":
		return "ignore"
	default:
		return "boot"
	}
}
