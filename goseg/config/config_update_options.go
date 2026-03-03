package config

import "groundseg/structs"

func WithPiers(piers []string) ConfUpdateOption {
	return func(patch *ConfPatch) {
		copied := append([]string(nil), piers...)
		patch.Piers = &copied
	}
}

func WithWgOn(enabled bool) ConfUpdateOption {
	return func(patch *ConfPatch) {
		patch.WgOn = &enabled
	}
}

func WithStartramReminderOne(reminded bool) ConfUpdateOption {
	return func(patch *ConfPatch) {
		patch.StartramReminderOne = &reminded
	}
}

func WithStartramReminderThree(reminded bool) ConfUpdateOption {
	return func(patch *ConfPatch) {
		patch.StartramReminderThree = &reminded
	}
}

func WithStartramReminderSeven(reminded bool) ConfUpdateOption {
	return func(patch *ConfPatch) {
		patch.StartramReminderSeven = &reminded
	}
}

func WithStartramReminderAll(reminded bool) ConfUpdateOption {
	return func(patch *ConfPatch) {
		patch.StartramReminderOne = &reminded
		patch.StartramReminderThree = &reminded
		patch.StartramReminderSeven = &reminded
	}
}

func WithPenpaiAllow(enabled bool) ConfUpdateOption {
	return func(patch *ConfPatch) {
		patch.PenpaiAllow = &enabled
	}
}

func WithGracefulExit(enabled bool) ConfUpdateOption {
	return func(patch *ConfPatch) {
		patch.GracefulExit = &enabled
	}
}

func WithSwapVal(value int) ConfUpdateOption {
	return func(patch *ConfPatch) {
		patch.SwapVal = &value
	}
}

func WithPenpaiRunning(running bool) ConfUpdateOption {
	return func(patch *ConfPatch) {
		patch.PenpaiRunning = &running
	}
}

func WithPenpaiActive(model string) ConfUpdateOption {
	return func(patch *ConfPatch) {
		patch.PenpaiActive = &model
	}
}

func WithPenpaiCores(cores int) ConfUpdateOption {
	return func(patch *ConfPatch) {
		patch.PenpaiCores = &cores
	}
}

func WithEndpointURL(endpoint string) ConfUpdateOption {
	return func(patch *ConfPatch) {
		patch.EndpointURL = &endpoint
	}
}

func WithWgRegistered(registered bool) ConfUpdateOption {
	return func(patch *ConfPatch) {
		patch.WgRegistered = &registered
	}
}

func WithRemoteBackupPassword(password string) ConfUpdateOption {
	return func(patch *ConfPatch) {
		patch.RemoteBackupPassword = &password
	}
}

func WithPubkey(pubkey string) ConfUpdateOption {
	return func(patch *ConfPatch) {
		patch.Pubkey = &pubkey
	}
}

func WithPrivkey(privkey string) ConfUpdateOption {
	return func(patch *ConfPatch) {
		patch.Privkey = &privkey
	}
}

func WithSalt(salt string) ConfUpdateOption {
	return func(patch *ConfPatch) {
		patch.Salt = &salt
	}
}

func WithKeyfile(path string) ConfUpdateOption {
	return func(patch *ConfPatch) {
		patch.AuthSessionPatch.KeyFile = &path
	}
}

func WithC2cInterval(seconds int) ConfUpdateOption {
	return func(patch *ConfPatch) {
		patch.C2cInterval = &seconds
	}
}

func WithSetup(step string) ConfUpdateOption {
	return func(patch *ConfPatch) {
		patch.Setup = &step
	}
}

func WithPwHash(hash string) ConfUpdateOption {
	return func(patch *ConfPatch) {
		patch.PwHash = &hash
	}
}

func WithAuthorizedSession(tokenID string, session structs.SessionInfo) ConfUpdateOption {
	return func(patch *ConfPatch) {
		if patch.AuthorizedSessions == nil {
			patch.AuthorizedSessions = make(map[string]structs.SessionInfo)
		}
		patch.AuthorizedSessions[tokenID] = session
	}
}

func WithUnauthorizedSession(tokenID string, session structs.SessionInfo) ConfUpdateOption {
	return func(patch *ConfPatch) {
		if patch.UnauthorizedSessions == nil {
			patch.UnauthorizedSessions = make(map[string]structs.SessionInfo)
		}
		patch.UnauthorizedSessions[tokenID] = session
	}
}

func WithDiskWarning(warning map[string]structs.DiskWarning) ConfUpdateOption {
	return func(patch *ConfPatch) {
		copied := make(map[string]structs.DiskWarning, len(warning))
		for key, val := range warning {
			copied[key] = val
		}
		patch.DiskWarning = &copied
	}
}

func WithLastKnownMDNS(url string) ConfUpdateOption {
	return func(patch *ConfPatch) {
		patch.LastKnownMDNS = &url
	}
}

func WithDisableSlsa(disable bool) ConfUpdateOption {
	return func(patch *ConfPatch) {
		patch.DisableSlsa = &disable
	}
}

func WithGSVersion(version string) ConfUpdateOption {
	return func(patch *ConfPatch) {
		patch.GSVersion = &version
	}
}

func WithBinHash(hash string) ConfUpdateOption {
	return func(patch *ConfPatch) {
		patch.BinHash = &hash
	}
}
