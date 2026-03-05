package config

import "groundseg/structs"

type ConfUpdateOption func(*ConfPatch)
type ConfigUpdateOption = ConfUpdateOption

type ConnectivityUpdateOption func(*ConnectivityPatch)
type RuntimeUpdateOption func(*RuntimePatch)
type StartramUpdateOption func(*StartramPatch)
type PenpaiUpdateOption func(*PenpaiPatch)
type AuthSessionUpdateOption func(*AuthSessionPatch)

type ConnectivityPatch struct {
	Piers                *[]string
	WgOn                 *bool
	NetCheck             *string
	UpdateMode           *string
	UpdateURL            *string
	UpdateBranch         *string
	RemoteBackupPassword *string
	C2CInterval          *int
	EndpointURL          *string
	ApiVersion           *string
	DiskWarning          *map[string]structs.DiskWarning
	WgRegistered         *bool
}

type RuntimePatch struct {
	GracefulExit   *bool
	SwapVal        *int
	SwapFile       *string
	Setup          *string
	LastKnownMDNS  *string
	LinuxUpdates   *linuxUpdatesPatch
	DockerData     *string
	GSVersion      *string
	CfgDir         *string
	UpdateInterval *int
	BinHash        *string
	Disable502     *bool
	SnapTime       *int
}

type StartramPatch struct {
	StartramReminderOne   *bool
	StartramReminderThree *bool
	StartramReminderSeven *bool
	Pubkey                *string
	Privkey               *string
	DisableSlsa           *bool
}

type PenpaiPatch struct {
	PenpaiAllow   *bool
	PenpaiRunning *bool
	PenpaiCores   *int
	PenpaiActive  *string
	PenpaiModels  []structs.Penpai
}

type AuthSessionPatch struct {
	PwHash               *string
	Salt                 *string
	KeyFile              *string
	AuthorizedSessions   map[string]structs.SessionInfo
	UnauthorizedSessions map[string]structs.SessionInfo
}

type linuxUpdatesPatch struct {
	Value    int    `json:"value"`
	Interval string `json:"interval"`
}

type ConfPatch struct {
	ConnectivityPatch
	RuntimePatch
	StartramPatch
	PenpaiPatch
	AuthSessionPatch
}

type ConfigPatch = ConfPatch

func WithConnectivityUpdates(options ...ConnectivityUpdateOption) ConfigUpdateOption {
	return func(patch *ConfPatch) {
		if patch == nil {
			return
		}
		for _, option := range options {
			if option != nil {
				option(&patch.ConnectivityPatch)
			}
		}
	}
}

func WithRuntimeUpdates(options ...RuntimeUpdateOption) ConfigUpdateOption {
	return func(patch *ConfPatch) {
		if patch == nil {
			return
		}
		for _, option := range options {
			if option != nil {
				option(&patch.RuntimePatch)
			}
		}
	}
}

func WithStartramUpdates(options ...StartramUpdateOption) ConfigUpdateOption {
	return func(patch *ConfPatch) {
		if patch == nil {
			return
		}
		for _, option := range options {
			if option != nil {
				option(&patch.StartramPatch)
			}
		}
	}
}

func WithPenpaiUpdates(options ...PenpaiUpdateOption) ConfigUpdateOption {
	return func(patch *ConfPatch) {
		if patch == nil {
			return
		}
		for _, option := range options {
			if option != nil {
				option(&patch.PenpaiPatch)
			}
		}
	}
}

func WithAuthSessionUpdates(options ...AuthSessionUpdateOption) ConfigUpdateOption {
	return func(patch *ConfPatch) {
		if patch == nil {
			return
		}
		for _, option := range options {
			if option != nil {
				option(&patch.AuthSessionPatch)
			}
		}
	}
}
