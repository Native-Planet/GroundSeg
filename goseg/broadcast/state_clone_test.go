package broadcast

import (
	"testing"

	"groundseg/structs"
)

func TestCloneBroadcastStateDetachesNestedCollections(t *testing.T) {
	original := structs.AuthBroadcast{
		Urbits: map[string]structs.Urbit{
			"~zod": {},
		},
	}
	urbit := original.Urbits["~zod"]
	urbit.Info.RemoteTlonBackups = []structs.BackupObject{{MD5: "a"}}
	original.Urbits["~zod"] = urbit
	original.System.Info.Drives = map[string]structs.SystemDrive{
		"disk0": {DriveID: 1},
	}
	original.System.Info.Wifi.Networks = []string{"Home"}

	cloned := cloneBroadcastState(original)
	cloned.Urbits["~zod"] = structs.Urbit{}
	cloned.System.Info.Drives["disk0"] = structs.SystemDrive{DriveID: 999}
	cloned.System.Info.Wifi.Networks[0] = "Mutated"

	if original.Urbits["~zod"].Info.RemoteTlonBackups[0].MD5 != "a" {
		t.Fatal("expected urbit backups to be deep-cloned")
	}
	if original.System.Info.Drives["disk0"].DriveID != 1 {
		t.Fatal("expected system drives map to be deep-cloned")
	}
	if original.System.Info.Wifi.Networks[0] != "Home" {
		t.Fatal("expected wifi network slice to be deep-cloned")
	}
}
