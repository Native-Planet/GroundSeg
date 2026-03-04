package transition

import "fmt"

// TransitionPublishPolicy controls whether transition publishing failures should fail the caller
// or be treated as best-effort delivery.
type TransitionPublishPolicy string

const (
	TransitionPublishStrict     TransitionPublishPolicy = "strict"
	TransitionPublishBestEffort TransitionPublishPolicy = "best_effort"
)

type TransitionPublishCriticality string

const (
	TransitionPublishCritical    TransitionPublishCriticality = "critical"
	TransitionPublishNonCritical TransitionPublishCriticality = "non_critical"
)

func TransitionPolicyForCriticality(criticality TransitionPublishCriticality) TransitionPublishPolicy {
	if criticality == TransitionPublishCritical {
		return TransitionPublishStrict
	}
	return TransitionPublishBestEffort
}

func HandleTransitionPublishError(context string, err error, policy TransitionPublishPolicy) error {
	if err == nil {
		return nil
	}
	if policy == TransitionPublishBestEffort {
		return nil
	}
	return fmt.Errorf("%s: %w", context, err)
}
