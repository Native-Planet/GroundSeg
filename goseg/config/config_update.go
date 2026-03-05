package config

func UpdateConfigTyped(opts ...ConfigUpdateOption) error {
	if len(opts) == 0 {
		return nil
	}

	patch := &ConfigPatch{}
	for _, opt := range opts {
		if opt != nil {
			opt(patch)
		}
	}
	if !patch.hasUpdates() {
		return nil
	}

	return updateConfigFromPatch(patch)
}

// UpdateConfTyped is a backwards-compatible alias for UpdateConfigTyped.
func UpdateConfTyped(opts ...ConfUpdateOption) error {
	return UpdateConfigTyped(opts...)
}

// UpdateConfig updates config using the map-style patch API.
func UpdateConfig(values map[string]interface{}) error {
	patch, err := buildConfigPatch(values)
	if err != nil {
		return err
	}
	return updateConfigFromPatch(patch)
}

// UpdateConf is a backwards-compatible alias for UpdateConfig.
func UpdateConf(values map[string]interface{}) error {
	return UpdateConfig(values)
}
