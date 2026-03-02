package config

func UpdateConfTyped(opts ...ConfUpdateOption) error {
	if len(opts) == 0 {
		return nil
	}

	patch := &ConfPatch{}
	for _, opt := range opts {
		if opt != nil {
			opt(patch)
		}
	}
	if !patch.hasUpdates() {
		return nil
	}

	return updateConfFromPatch(patch)
}

// UpdateConf updates config using the map-style patch API.
func UpdateConf(values map[string]interface{}) error {
	patch, err := buildConfigPatch(values)
	if err != nil {
		return err
	}
	return updateConfFromPatch(patch)
}
