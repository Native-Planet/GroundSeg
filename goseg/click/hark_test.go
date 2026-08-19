package click

import (
	"strings"
	"testing"
)

func TestHarkNotificationHoonUsesCurrentAction(t *testing.T) {
	hoon := harkNotificationHoon("disk-space", "'Drive warning'")

	for _, want := range []string{
		"(poke [our.bowl %hark] %hark-action",
		"[%add-yarn & & id rope now.bowl content / ~]",
		"[~ ~ %nativeplanet /nativeplanet/disk-space]",
		"~['Drive warning']",
	} {
		if !strings.Contains(hoon, want) {
			t.Fatalf("generated Hoon does not contain %q:\n%s", want, hoon)
		}
	}

	for _, retired := range []string{"%hark-store", "%add-note"} {
		if strings.Contains(hoon, retired) {
			t.Fatalf("generated Hoon still contains retired action %q:\n%s", retired, hoon)
		}
	}
}
