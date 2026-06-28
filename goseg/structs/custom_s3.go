package structs

import "strings"

func LegacyCustomS3Domain(conf UrbitDocker) string {
	if domain := strings.TrimSpace(conf.CustomS3WebLocal); domain != "" {
		return domain
	}
	if domain := strings.TrimSpace(conf.CustomS3Web); domain != "" {
		return domain
	}
	return strings.TrimSpace(conf.CustomS3WebRemote)
}

func SyncCustomS3Domains(conf *UrbitDocker) {
	legacyDomain := strings.TrimSpace(conf.CustomS3Web)
	if legacyDomain != "" && strings.TrimSpace(conf.CustomS3WebLocal) == "" && strings.TrimSpace(conf.CustomS3WebRemote) == "" {
		if strings.TrimSpace(conf.CustomS3WebLocal) == "" {
			conf.CustomS3WebLocal = legacyDomain
		}
		if strings.TrimSpace(conf.CustomS3WebRemote) == "" {
			conf.CustomS3WebRemote = legacyDomain
		}
	}
	conf.CustomS3Web = LegacyCustomS3Domain(*conf)
}
