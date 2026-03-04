package seams

// CallbackRequirements validates that a named set of callbacks/dependencies is provided.
type CallbackRequirements struct {
	required   []string
	configured map[string]bool
}

func NewCallbackRequirements(required ...string) CallbackRequirements {
	return CallbackRequirements{
		required:   append([]string(nil), required...),
		configured: map[string]bool{},
	}
}

func (requirements CallbackRequirements) With(name string, configured bool) CallbackRequirements {
	if configured {
		requirements.configured[name] = true
	}
	return requirements
}

func (requirements CallbackRequirements) Missing() []string {
	missing := []string{}
	for _, name := range requirements.required {
		if !requirements.configured[name] {
			missing = append(missing, name)
		}
	}
	return missing
}
