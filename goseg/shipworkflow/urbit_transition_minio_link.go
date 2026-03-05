package shipworkflow

import (
	"fmt"

	"groundseg/click"
	"groundseg/docker"
	dockerOrchestration "groundseg/docker/orchestration"
	"groundseg/structs"
)

func buildToggleMinIOLinkSteps(patp string, _ structs.WsUrbitPayload) []transitionStep[string] {
	coordinator := minIOLinkTransitionCoordinator{patp: patp}
	return coordinator.steps()
}

type minIOLinkTransitionCoordinator struct {
	patp      string
	linkState minIOLinkState
}

func (c *minIOLinkTransitionCoordinator) steps() []transitionStep[string] {
	return []transitionStep[string]{
		{
			Run: c.loadCurrentState,
		},
		{
			Event: "unlinking",
			EmitWhen: func() bool {
				return c.linkState.IsLinked
			},
			Run: c.unlinkIfNeeded,
		},
		{
			Event: "unlink-success",
			EmitWhen: func() bool {
				return c.linkState.IsLinked
			},
		},
		{
			Event: "linking",
			EmitWhen: func() bool {
				return !c.linkState.IsLinked
			},
			Run: c.linkIfNeeded,
		},
		{
			Event: "success",
			EmitWhen: func() bool {
				return !c.linkState.IsLinked
			},
		},
	}
}

func (c *minIOLinkTransitionCoordinator) loadCurrentState() error {
	state, stateErr := loadMinIOLinkState(c.patp)
	if stateErr != nil {
		return stateErr
	}
	c.linkState = state
	return nil
}

func (c *minIOLinkTransitionCoordinator) unlinkIfNeeded() error {
	if !c.linkState.IsLinked {
		return nil
	}
	return unbindMinIOLink(c.patp)
}

func (c *minIOLinkTransitionCoordinator) linkIfNeeded() error {
	if c.linkState.IsLinked {
		return nil
	}
	return linkMinIOLink(c.patp, c.linkState)
}

type minIOLinkState struct {
	IsLinked bool
	Endpoint string
}

func loadMinIOLinkState(patp string) (minIOLinkState, error) {
	shipConf := getUrbitConfigFn(patp)
	endpoint := shipConf.CustomS3Web
	if endpoint == "" {
		endpoint = fmt.Sprintf("s3.%s", shipConf.WgURL)
	}
	return minIOLinkState{
		IsLinked: shipConf.MinIOLinked,
		Endpoint: endpoint,
	}, nil
}

func unbindMinIOLink(patp string) error {
	if err := click.UnlinkStorage(patp); err != nil {
		return fmt.Errorf("failed to unlink minio information %s: %w", patp, err)
	}
	if err := persistShipUrbitSectionConfig[structs.UrbitFeatureConfig](patp, dockerOrchestration.UrbitConfigSectionFeature, func(conf *structs.UrbitFeatureConfig) error {
		conf.MinIOLinked = false
		return nil
	}); err != nil {
		return fmt.Errorf("could not update urbit config: %w", err)
	}
	return nil
}

func linkMinIOLink(patp string, state minIOLinkState) error {
	svcAccount, err := docker.CreateMinIOServiceAccount(patp)
	if err != nil {
		return fmt.Errorf("failed to create minio service account for %s: %w", patp, err)
	}
	if err := click.LinkStorage(patp, state.Endpoint, svcAccount); err != nil {
		return fmt.Errorf("failed to link minio information %s: %w", patp, err)
	}
	if err := persistShipUrbitSectionConfig[structs.UrbitFeatureConfig](patp, dockerOrchestration.UrbitConfigSectionFeature, func(conf *structs.UrbitFeatureConfig) error {
		conf.MinIOLinked = true
		return nil
	}); err != nil {
		return fmt.Errorf("could not update urbit config: %w", err)
	}
	return nil
}
