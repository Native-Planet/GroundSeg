package startram

import (
	"goseg/config"
	"goseg/structs"
	"strings"
)

// compare latest urbit config struct to retrieve config
// return 'true' if something changes
func RectifyUrbit(patp string) (bool, error) {
	modified := false
	startramConfig := config.StartramConfig // a structs.StartramRetrieve
	config.LoadUrbitConfig(patp)
	local := config.UrbitConf(patp) // a structs.UrbitDocker
	for _, remote := range startramConfig.Subdomains {
		// for urbit web
		subd := strings.Split(remote.URL, ".")[0]
		if subd == patp && remote.SvcType == "urbit-web" && remote.Status == "ok" {
			// update alias
			if remote.Alias != "null" && remote.Alias != local.CustomUrbitWeb {
				local.CustomUrbitWeb = remote.Alias
				modified = true
			}
			// update www port
			if remote.Port != local.WgHTTPPort {
				local.WgHTTPPort = remote.Port
				modified = true
			}
			// update remote url
			if remote.URL != local.WgURL {
				local.WgURL = remote.URL
				modified = true
			}
			continue
		}
		// for urbit ames
		nestd := strings.Join(strings.Split(remote.URL, ".")[:2], ".")
		if nestd == "ames."+patp && remote.SvcType == "urbit-ames" && remote.Status == "ok" {
			if remote.Port != local.WgAmesPort {
				local.WgAmesPort = remote.Port
				modified = true
			}
			continue
		}
		// for minio
		if nestd == "s3."+patp && remote.SvcType == "minio" && remote.Status == "ok" {
			if remote.Port != local.WgS3Port {
				local.WgS3Port = remote.Port
				modified = true
			}
			continue
		}
		// for minio console
		consd := strings.Join(strings.Split(remote.URL, ".")[:3], ".")
		if consd == "console.s3."+patp && remote.SvcType == "minio-console" && remote.Status == "ok" {
			if remote.Port != local.WgConsolePort {
				local.WgConsolePort = remote.Port
				modified = true
			}
			continue
		}
	}
	if modified {
		config.UpdateUrbitConfig(map[string]structs.UrbitDocker{patp: local})
	}
	return modified, nil
}
