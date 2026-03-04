package backup

import (
	"groundseg/config"
	"groundseg/structs"
	"testing"
	"time"
)

func TestGenerateTimeOfDayDeterministic(t *testing.T) {
	a := GenerateTimeOfDay("~zod")
	b := GenerateTimeOfDay("~zod")
	if !a.Equal(b) {
		t.Fatalf("expected deterministic output for same seed: %v vs %v", a, b)
	}
}

func TestGenerateTimeOfDayRange(t *testing.T) {
	got := GenerateTimeOfDay("~nec")
	if got.Hour() < 0 || got.Hour() > 23 {
		t.Fatalf("hour out of range: %d", got.Hour())
	}
	if got.Minute() < 0 || got.Minute() > 59 {
		t.Fatalf("minute out of range: %d", got.Minute())
	}
	if got.Second() < 0 || got.Second() > 59 {
		t.Fatalf("second out of range: %d", got.Second())
	}
}

func TestRunRemoteBackupPassWaitsForStartramSnapshot(t *testing.T) {
	resetBackupSeamsForTest(t)

	snapshotWaited := false
	GetStartramConfigSnapshotForRoutine = func() config.StartramConfigSnapshot {
		return config.StartramConfigSnapshot{}
	}
	SleepForRoutine = func(d time.Duration) {
		snapshotWaited = true
		if d != 30*time.Second {
			t.Fatalf("unexpected snapshot wait duration: %v", d)
		}
	}

	RunRemoteBackupPass()
	if !snapshotWaited {
		t.Fatal("expected snapshot wait when no fresh config snapshot is available")
	}
}

func TestRunRemoteBackupPassUploadsWhenScheduled(t *testing.T) {
	resetBackupSeamsForTest(t)

	window := GenerateTimeOfDay("seed-remote-pass")
	uploadCalls := 0

	GetStartramConfigSnapshotForRoutine = func() config.StartramConfigSnapshot {
		return config.StartramConfigSnapshot{
			Fresh: true,
			Value: structs.StartramRetrieve{
				UrlID: "seed-remote-pass",
			},
		}
	}
	ConfForRoutine = func() structs.SysConfig {
		return structs.SysConfig{
			Connectivity: structs.ConnectivityConfig{
				Piers:                []string{"~zod"},
				WgRegistered:         true,
				RemoteBackupPassword: "pw",
			},
		}
	}
	UploadLatestBackupForRoutine = func(patp, password, backupDir string) error {
		uploadCalls++
		if patp != "~zod" {
			t.Fatalf("unexpected ship %q", patp)
		}
		if password != "pw" {
			t.Fatalf("unexpected password %q", password)
		}
		if backupDir == "" {
			t.Fatalf("expected backup dir")
		}
		return nil
	}
	NowForRoutine = func() time.Time { return window }

	ResetRemoteBackupStateForTest()
	RunRemoteBackupPass()
	if uploadCalls == 0 {
		t.Fatal("expected backup upload to run during scheduled window")
	}
}

func TestRunLocalBackupPassCreatesBackupForStaleSnapshot(t *testing.T) {
	resetBackupSeamsForTest(t)

	localBackupCalls := 0
	ConfForRoutine = func() structs.SysConfig {
		return structs.SysConfig{Connectivity: structs.ConnectivityConfig{Piers: []string{"~zod"}}}
	}
	UrbitConfForRoutine = func(string) structs.UrbitDocker {
		return structs.UrbitDocker{
			UrbitBackupConfig: structs.UrbitBackupConfig{
				BackupTime: "0100",
			},
		}
	}
	mostRecent := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC).Add(-48 * time.Hour)
	LatestDailyBackupForRoutine = func(_, _ string) (time.Time, error) {
		return mostRecent, nil
	}
	CreateLocalBackupForRoutine = func(_ string, _ string) error {
		localBackupCalls++
		return nil
	}
	NowForRoutine = func() time.Time {
		return time.Date(2020, 1, 1, 2, 0, 0, 0, time.UTC)
	}

	RunLocalBackupPass()
	if localBackupCalls != 1 {
		t.Fatalf("expected local backup to run once, got %d", localBackupCalls)
	}
}

func TestRunLocalBackupPassSkipsWhenRecent(t *testing.T) {
	resetBackupSeamsForTest(t)

	localBackupCalls := 0
	ConfForRoutine = func() structs.SysConfig {
		return structs.SysConfig{Connectivity: structs.ConnectivityConfig{Piers: []string{"~zod"}}}
	}
	UrbitConfForRoutine = func(string) structs.UrbitDocker {
		return structs.UrbitDocker{
			UrbitBackupConfig: structs.UrbitBackupConfig{
				BackupTime: "2359",
			},
		}
	}
	recent := time.Date(2020, 1, 1, 23, 58, 0, 0, time.UTC)
	LatestDailyBackupForRoutine = func(_, _ string) (time.Time, error) {
		return recent, nil
	}
	CreateLocalBackupForRoutine = func(string, string) error {
		localBackupCalls++
		return nil
	}
	NowForRoutine = func() time.Time {
		return time.Date(2020, 1, 1, 23, 59, 0, 0, time.UTC)
	}

	RunLocalBackupPass()
	if localBackupCalls != 0 {
		t.Fatalf("expected local backup to be skipped when most recent backup is current")
	}
}

func resetBackupSeamsForTest(t *testing.T) {
	t.Helper()
	origSnapshot := GetStartramConfigSnapshotForRoutine
	origConf := ConfForRoutine
	origUrbit := UrbitConfForRoutine
	origUpload := UploadLatestBackupForRoutine
	origLatest := LatestDailyBackupForRoutine
	origCreate := CreateLocalBackupForRoutine
	origSleep := SleepForRoutine
	origNow := NowForRoutine

	t.Cleanup(func() {
		GetStartramConfigSnapshotForRoutine = origSnapshot
		ConfForRoutine = origConf
		UrbitConfForRoutine = origUrbit
		UploadLatestBackupForRoutine = origUpload
		LatestDailyBackupForRoutine = origLatest
		CreateLocalBackupForRoutine = origCreate
		SleepForRoutine = origSleep
		NowForRoutine = origNow
		ResetRemoteBackupStateForTest()
	})
}
