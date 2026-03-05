package shipworkflow

import (
	"fmt"
	"strings"

	"groundseg/broadcast"
	"groundseg/docker"
	dockerOrchestration "groundseg/docker/orchestration"
	"groundseg/startram"
	"groundseg/structs"
	"groundseg/transition"
)

type aliasTransitionPolicy struct {
	oldDomainFn    func(structs.UrbitDocker) string
	aliasSubjectFn func(string) string
	applyFn        func(*structs.UrbitWebConfig, string)
	invalidLabel   string
	errorLabel     string
}

func toggleAlias(patp string) error {
	currentConf := getUrbitConfigFn(patp)
	nextShowUrbitWeb := "custom"
	if currentConf.ShowUrbitWeb == "custom" {
		nextShowUrbitWeb = "default"
	}
	if err := persistShipUrbitSectionConfig[structs.UrbitWebConfig](patp, dockerOrchestration.UrbitConfigSectionWeb, func(conf *structs.UrbitWebConfig) error {
		conf.ShowUrbitWeb = nextShowUrbitWeb
		return nil
	}); err != nil {
		return fmt.Errorf("could not update urbit config: %w", err)
	}
	return nil
}

func toggleBootStatus(patp string) error {
	shipConf := getUrbitConfigFn(patp)
	nextBootStatus := "ignore"
	if shipConf.BootStatus == "ignore" {
		statusMap, err := docker.GetShipStatus([]string{patp})
		if err != nil {
			return fmt.Errorf("failed to get ship status for %s: %w", patp, err)
		}
		status, exists := statusMap[patp]
		if !exists {
			return fmt.Errorf("running status for %s doesn't exist: %w", patp, errShipStatusNotFound)
		}
		if strings.Contains(status, "Up") {
			nextBootStatus = "boot"
		} else {
			nextBootStatus = "noboot"
		}
	}
	if err := persistShipUrbitSectionConfig[structs.UrbitRuntimeConfig](patp, dockerOrchestration.UrbitConfigSectionRuntime, func(conf *structs.UrbitRuntimeConfig) error {
		conf.BootStatus = nextBootStatus
		return nil
	}); err != nil {
		return fmt.Errorf("could not update urbit config: %w", err)
	}
	return nil
}

func toggleAutoReboot(patp string) error {
	if err := loadUrbitConfigFn(patp); err != nil {
		return fmt.Errorf("failed to load fresh urbit config: %w", err)
	}
	currentConf := getUrbitConfigFn(patp)
	if err := persistShipUrbitSectionConfig[structs.UrbitFeatureConfig](patp, dockerOrchestration.UrbitConfigSectionFeature, func(conf *structs.UrbitFeatureConfig) error {
		conf.DisableShipRestarts = !currentConf.DisableShipRestarts
		return nil
	}); err != nil {
		return fmt.Errorf("could not update urbit config: %w", err)
	}
	if err := broadcast.BroadcastToClients(); err != nil {
		return transition.HandleTransitionPublishError(
			fmt.Sprintf("publish auto-reboot toggle transition for %s", patp),
			err,
			transition.TransitionPolicyForCriticality(transition.TransitionPublishNonCritical),
		)
	}
	return nil
}

func buildAliasTransitionSteps(patp string, payload structs.WsUrbitPayload, policy aliasTransitionPolicy) []transitionStep[string] {
	return []transitionStep[string]{
		{
			Event: "success",
			Run: func() error {
				alias := payload.Payload.Domain
				currentConf := getUrbitConfigFn(patp)
				oldDomain := policy.oldDomainFn(currentConf)
				areAliases, err := AreSubdomainsAliases(alias, oldDomain)
				if err != nil {
					return fmt.Errorf("failed to check %s alias for %s: %w", policy.errorLabel, patp, err)
				}
				if !areAliases {
					return fmt.Errorf("invalid %s alias for %s", policy.invalidLabel, patp)
				}
				if err := startram.AliasCreate(policy.aliasSubjectFn(patp), alias); err != nil {
					return fmt.Errorf("set %s alias for %s: %w", policy.errorLabel, patp, err)
				}
				if err := persistShipUrbitSectionConfig[structs.UrbitWebConfig](patp, dockerOrchestration.UrbitConfigSectionWeb, func(conf *structs.UrbitWebConfig) error {
					policy.applyFn(conf, alias)
					return nil
				}); err != nil {
					return fmt.Errorf("could not update urbit config: %w", err)
				}
				return nil
			},
		},
	}
}

func buildUrbitDomainSteps(patp string, payload structs.WsUrbitPayload) []transitionStep[string] {
	return buildAliasTransitionSteps(patp, payload, aliasTransitionPolicy{
		oldDomainFn:    func(conf structs.UrbitDocker) string { return conf.WgURL },
		aliasSubjectFn: func(patp string) string { return patp },
		applyFn: func(conf *structs.UrbitWebConfig, alias string) {
			conf.CustomUrbitWeb = alias
			conf.ShowUrbitWeb = "custom"
		},
		invalidLabel: "urbit domain",
		errorLabel:   "urbit domain",
	})
}

func buildMinIODomainSteps(patp string, payload structs.WsUrbitPayload) []transitionStep[string] {
	return buildAliasTransitionSteps(patp, payload, aliasTransitionPolicy{
		oldDomainFn: func(conf structs.UrbitDocker) string {
			return fmt.Sprintf("s3.%s", conf.WgURL)
		},
		aliasSubjectFn: func(patp string) string {
			return fmt.Sprintf("s3.%s", patp)
		},
		applyFn: func(conf *structs.UrbitWebConfig, alias string) {
			conf.CustomS3Web = alias
		},
		invalidLabel: "minio domain",
		errorLabel:   "minio domain",
	})
}
