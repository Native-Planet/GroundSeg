package orchestration

import (
	"testing"

	"groundseg/transition"
)

func TestContainerConfigForTypeDispatchesKnownTypes(t *testing.T) {
	cases := []struct {
		containerType transition.ContainerType
	}{
		{transition.ContainerTypeVere},
		{transition.ContainerTypeNetdata},
		{transition.ContainerTypeMinio},
		{transition.ContainerTypeMinioMC},
		{transition.ContainerTypeWireguard},
		{transition.ContainerTypeLlamaAPI},
	}

	for _, tc := range cases {
		if _, ok := containerConfigBuilders[tc.containerType]; !ok {
			t.Fatalf("container type %s should be registered", tc.containerType)
		}
	}
}

func TestContainerConfigForTypeRejectsUnknownType(t *testing.T) {
	_, _, err := ContainerConfigForTypeString("~zod", "unknown")
	if err == nil {
		t.Fatal("expected unknown container type to fail")
	}
}
