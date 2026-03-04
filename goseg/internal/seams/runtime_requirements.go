package seams

import (
	"fmt"
	"reflect"
	"strings"
)

// TaggedCallbackRequirements provides reusable validation across runtime "callback bag" structs.
// Fields can declare one or more callback groups via struct tags, for example:
// `runtime:"startup"` or `runtime:"startup,optional"`.
//
// Nested structs are traversed recursively, which allows embedded runtime role blocks
// while still keeping group-level validation.
type TaggedCallbackRequirements struct {
	requiredGroups map[string]struct{}
}

func NewCallbackRequirementsWithGroups(groups ...string) TaggedCallbackRequirements {
	required := make(map[string]struct{}, len(groups))
	for _, group := range groups {
		required[strings.TrimSpace(group)] = struct{}{}
	}
	return TaggedCallbackRequirements{
		requiredGroups: required,
	}
}

func (requirements TaggedCallbackRequirements) ValidateCallbacks(runtime interface{}, runtimeName string) error {
	missing := MissingTaggedCallbacks(runtime, requirements.requiredGroups)
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("%s missing required callbacks: %s", runtimeName, strings.Join(missing, ", "))
}

// MissingTaggedCallbacks returns missing callback names for fields with matching `runtime` tags.
func MissingTaggedCallbacks(runtime any, requiredGroups map[string]struct{}) []string {
	return missingTaggedCallbacks(reflect.ValueOf(runtime), requiredGroups)
}

func missingTaggedCallbacks(value reflect.Value, requiredGroups map[string]struct{}) []string {
	if !value.IsValid() {
		return nil
	}
	if value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return nil
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return nil
	}

	missing := []string{}
	for i := 0; i < value.NumField(); i++ {
		fieldType := value.Type().Field(i)
		fieldValue := value.Field(i)

		if fieldType.Anonymous {
			missing = append(missing, missingTaggedCallbacks(fieldValue, requiredGroups)...)
			continue
		}

		tag := fieldType.Tag.Get("runtime")
		if tag == "" {
			continue
		}
		if !requiredGroupsMatch(tag, requiredGroups) {
			continue
		}
		if !isConfigured(fieldValue) {
			name := fieldType.Tag.Get("runtime_name")
			if name == "" {
				name = fieldType.Name
			}
			missing = append(missing, name)
		}
	}
	return missing
}

func requiredGroupsMatch(tag string, requiredGroups map[string]struct{}) bool {
	if len(requiredGroups) == 0 {
		return true
	}
	for _, group := range strings.Split(tag, ",") {
		group = strings.TrimSpace(group)
		if _, ok := requiredGroups[group]; ok {
			return true
		}
	}
	return false
}

func isConfigured(value reflect.Value) bool {
	switch value.Kind() {
	case reflect.Func, reflect.Interface, reflect.Chan, reflect.Map, reflect.Ptr, reflect.Slice:
		return !value.IsNil()
	default:
		return value.IsValid() && !value.IsZero()
	}
}
