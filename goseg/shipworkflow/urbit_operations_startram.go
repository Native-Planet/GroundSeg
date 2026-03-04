package shipworkflow

import (
	"fmt"
	"strings"

	"groundseg/startram"
	"groundseg/structs"
)

func startramReminder(patp string, remind bool) error {
	if err := persistShipUrbitFeatureConfig(patp, func(conf *structs.UrbitFeatureConfig) error {
		conf.StartramReminder = remind
		return nil
	}); err != nil {
		return fmt.Errorf("Couldn't update urbit config: %w", err)
	}
	return nil
}

func urbitDeleteStartramService(patp string, service string) error {
	settings := getStartramSettingsSnapshot()
	parts := strings.Split(settings.EndpointURL, ".")
	if len(parts) < 2 {
		return fmt.Errorf("Failed to recreate subdomain for manual service deletion")
	}

	baseURL := parts[len(parts)-2] + "." + parts[len(parts)-1]
	var subdomain string
	switch service {
	case "urbit-web":
		subdomain = fmt.Sprintf("%s.%s", patp, baseURL)
	case "urbit-ames":
		subdomain = fmt.Sprintf("%s.%s.%s", "ames", patp, baseURL)
	case "minio":
		subdomain = fmt.Sprintf("%s.%s.%s", "s3", patp, baseURL)
	case "minio-console":
		subdomain = fmt.Sprintf("%s.%s.%s", "console.s3", patp, baseURL)
	case "minio-bucket":
		subdomain = fmt.Sprintf("%s.%s.%s", "bucket.s3", patp, baseURL)
	default:
		return fmt.Errorf("Invalid service type: unable to manually delete service")
	}
	if err := startram.SvcDelete(subdomain, service); err != nil {
		return fmt.Errorf("Failed to delete startram service: %w", err)
	}
	_, err := startram.SyncRetrieve()
	if err != nil {
		return fmt.Errorf("Failed to retrieve after manual service deletion: %w", err)
	}
	return nil
}
