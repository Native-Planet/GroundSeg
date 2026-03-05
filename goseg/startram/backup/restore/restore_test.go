package restore

import (
	"errors"
	"testing"

	"groundseg/config"
	"groundseg/structs"
)

func TestRestoreBackupProdRemoteFlow(t *testing.T) {
	t.Parallel()

	var fetchedRemote bool
	var persistedRemote bool
	var mountedBase bool
	var wroteVolume bool
	var committedDesk bool
	var restoredTlon bool

	runtime := RestoreRuntime{
		FetchRemoteFn: func(ship string, timestamp int, md5hash string) ([]byte, error) {
			if ship != "~zod" || timestamp != 101 || md5hash != "abc123" {
				t.Fatalf("unexpected remote fetch args: ship=%s timestamp=%d md5=%s", ship, timestamp, md5hash)
			}
			fetchedRemote = true
			return []byte("remote-backup"), nil
		},
		PersistRemoteFn: func(ship string, timestamp int, data []byte) error {
			if ship != "~zod" || timestamp != 101 || string(data) != "remote-backup" {
				t.Fatalf("unexpected persist args: ship=%s timestamp=%d data=%q", ship, timestamp, string(data))
			}
			persistedRemote = true
			return nil
		},
		FetchLocalFn: func(string, string, string, int) ([]byte, error) {
			t.Fatal("FetchLocalFn should not be called for remote source")
			return nil, errors.New("unreachable")
		},
		MountBaseDeskFn: func(string) error {
			mountedBase = true
			return nil
		},
		WriteToVolumeFn: func(ship string, data []byte) error {
			if ship != "~zod" || string(data) != "remote-backup" {
				t.Fatalf("unexpected write args: ship=%s data=%q", ship, string(data))
			}
			wroteVolume = true
			return nil
		},
		CommitDeskFn: func(ship string, desk string) error {
			if ship != "~zod" || desk != "base" {
				t.Fatalf("unexpected commit args: ship=%s desk=%s", ship, desk)
			}
			committedDesk = true
			return nil
		},
		RestoreTlonFn: func(ship string) error {
			if ship != "~zod" {
				t.Fatalf("unexpected restore ship: %s", ship)
			}
			restoredTlon = true
			return nil
		},
		GetBasePathFn: func() config.RuntimeContext {
			return config.RuntimeContext{BasePath: "/unused-for-remote"}
		},
	}

	err := RestoreBackupProd(runtime, RestoreBackupRequest{
		Ship:      "~zod",
		Timestamp: 101,
		MD5Hash:   "abc123",
		Source:    SourceRemote,
	})
	if err != nil {
		t.Fatalf("RestoreBackupProd returned error: %v", err)
	}
	if !fetchedRemote || !persistedRemote || !wroteVolume || !committedDesk || !restoredTlon {
		t.Fatalf("expected full remote restore flow to run, got remote=%v persist=%v write=%v commit=%v restore=%v",
			fetchedRemote, persistedRemote, wroteVolume, committedDesk, restoredTlon)
	}
	if mountedBase {
		t.Fatal("MountBaseDeskFn should not run for remote source")
	}
}

func TestRestoreBackupProdLocalFlowMountsBeforeWrite(t *testing.T) {
	t.Parallel()

	mounted := false
	wrote := false

	runtime := RestoreRuntime{
		FetchRemoteFn: func(string, int, string) ([]byte, error) {
			t.Fatal("FetchRemoteFn should not be called for local source")
			return nil, errors.New("unreachable")
		},
		PersistRemoteFn: func(string, int, []byte) error {
			t.Fatal("PersistRemoteFn should not be called for local source")
			return errors.New("unreachable")
		},
		FetchLocalFn: func(basePath string, ship string, backupType string, timestamp int) ([]byte, error) {
			if basePath != "/var/lib/groundseg" || ship != "~bus" || backupType != "weekly" || timestamp != 202 {
				t.Fatalf("unexpected local fetch args: base=%s ship=%s type=%s ts=%d", basePath, ship, backupType, timestamp)
			}
			return []byte("local-backup"), nil
		},
		MountBaseDeskFn: func(ship string) error {
			if ship != "~bus" {
				t.Fatalf("unexpected ship for mount: %s", ship)
			}
			mounted = true
			return nil
		},
		WriteToVolumeFn: func(ship string, data []byte) error {
			if !mounted {
				t.Fatal("expected mount to happen before write")
			}
			if ship != "~bus" || string(data) != "local-backup" {
				t.Fatalf("unexpected write args: ship=%s data=%q", ship, string(data))
			}
			wrote = true
			return nil
		},
		CommitDeskFn: func(string, string) error { return nil },
		RestoreTlonFn: func(string) error {
			return nil
		},
		GetBasePathFn: func() config.RuntimeContext {
			return config.RuntimeContext{BasePath: "/var/lib/groundseg"}
		},
	}

	err := RestoreBackupProd(runtime, RestoreBackupRequest{
		Ship:            "~bus",
		Timestamp:       202,
		LocalBackupType: "weekly",
		Source:          SourceLocal,
	})
	if err != nil {
		t.Fatalf("RestoreBackupProd local flow returned error: %v", err)
	}
	if !wrote {
		t.Fatal("expected local restore payload to be written")
	}
}

func TestRestoreBackupProdRejectsUnsupportedSource(t *testing.T) {
	t.Parallel()

	err := RestoreBackupProd(RestoreRuntime{}, RestoreBackupRequest{
		Ship:   "~zod",
		Source: "unknown",
	})
	if err == nil {
		t.Fatal("expected unsupported source error")
	}
}

func TestRestoreBackupDevUsesHighestTimestamp(t *testing.T) {
	t.Parallel()

	var persistedTimestamp int

	runtime := RestoreDevRuntime{
		FetchConfigFn: func() (structs.StartramRetrieve, error) {
			return structs.StartramRetrieve{
				Backups: []structs.Backup{
					{
						"~zod": []structs.BackupObject{
							{Timestamp: 1, MD5: "md5-1"},
							{Timestamp: 9, MD5: "md5-9"},
							{Timestamp: 3, MD5: "md5-3"},
						},
					},
				},
			}, nil
		},
		FetchRemoteFn: func(ship string, timestamp int, md5hash string) ([]byte, error) {
			if ship != "~zod" || timestamp != 9 || md5hash != "md5-9" {
				t.Fatalf("unexpected remote fetch args: ship=%s timestamp=%d md5=%s", ship, timestamp, md5hash)
			}
			return []byte("restored"), nil
		},
		PersistRemoteFn: func(ship string, timestamp int, data []byte) error {
			if ship != "~zod" || string(data) != "restored" {
				t.Fatalf("unexpected persist args: ship=%s data=%q", ship, string(data))
			}
			persistedTimestamp = timestamp
			return nil
		},
	}

	if err := RestoreBackupDev(runtime, "~zod"); err != nil {
		t.Fatalf("RestoreBackupDev returned error: %v", err)
	}
	if persistedTimestamp != 9 {
		t.Fatalf("expected highest timestamp to persist, got %d", persistedTimestamp)
	}
}

func TestRestoreBackupDevReturnsErrorWhenNoBackupFound(t *testing.T) {
	t.Parallel()

	runtime := RestoreDevRuntime{
		FetchConfigFn: func() (structs.StartramRetrieve, error) {
			return structs.StartramRetrieve{Backups: []structs.Backup{{"~other": []structs.BackupObject{{Timestamp: 4, MD5: "x"}}}}}, nil
		},
		FetchRemoteFn: func(string, int, string) ([]byte, error) {
			t.Fatal("unexpected remote fetch when no backup exists for target ship")
			return nil, errors.New("unreachable")
		},
		PersistRemoteFn: func(string, int, []byte) error {
			t.Fatal("unexpected persist when no backup exists for target ship")
			return errors.New("unreachable")
		},
	}

	err := RestoreBackupDev(runtime, "~zod")
	if err == nil {
		t.Fatal("expected RestoreBackupDev to fail when target ship has no backups")
	}
}
