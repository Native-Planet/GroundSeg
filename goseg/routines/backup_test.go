package routines

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"groundseg/config"
	"groundseg/structs"
)

func TestScheduledBackupsHonorShipSettings(t *testing.T) {
	saved := config.UrbitsConfig
	t.Cleanup(func() { config.UrbitsConfig = saved })
	config.UrbitsConfig = map[string]structs.UrbitDocker{
		"zod": {},
		"nec": {LocalTlonBackup: true},
		"bud": {RemoteTlonBackup: true},
		"wes": {LocalTlonBackup: true, RemoteTlonBackup: true},
	}
	piers := []string{"zod", "nec", "bud", "wes"}
	root := t.TempDir()

	var created []string
	createScheduledBackups(root, piers, func(ship, daily, weekly, monthly string) error {
		created = append(created, ship)
		return nil
	})
	if !slices.Equal(created, []string{"nec", "wes"}) {
		t.Errorf("local backup calls = %v, want [nec wes]", created)
	}
	for _, ship := range []string{"zod", "bud"} {
		if _, err := os.Stat(filepath.Join(root, ship)); !os.IsNotExist(err) {
			t.Errorf("expected no local backup directory for %s, got %v", ship, err)
		}
	}

	// An existing daily archive makes every ship a candidate for upload.
	for _, ship := range piers {
		daily := filepath.Join(root, ship, "daily")
		if err := os.MkdirAll(daily, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(daily, "100"), []byte("backup"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	var uploaded []string
	uploadScheduledBackups(root, piers, "password", func(ship, password, file string) error {
		uploaded = append(uploaded, ship)
		return nil
	})
	if !slices.Equal(uploaded, []string{"bud", "wes"}) {
		t.Errorf("remote upload calls = %v, want [bud wes]", uploaded)
	}
}

func TestRemoteBackupDailySchedule(t *testing.T) {
	now := time.Date(2026, 11, 16, 23, 30, 0, 0, time.FixedZone("CST", -6*60*60))
	scheduled := generateTimeOfDay("test-installation", now)
	if scheduled.Year() != 2026 || scheduled.Month() != time.November || scheduled.Day() != 17 || scheduled.Location() != time.UTC {
		t.Fatalf("schedule must use the current UTC date, got %v", scheduled)
	}
	next := generateTimeOfDay("test-installation", now.AddDate(0, 0, 1))
	if !next.Equal(scheduled.AddDate(0, 0, 1)) {
		t.Fatalf("daily schedule changed time of day: %v -> %v", scheduled, next)
	}
	if !remoteBackupDue(scheduled, time.Time{}, scheduled.Add(30*time.Second)) {
		t.Fatal("minute polling must trigger the scheduled backup")
	}
	if remoteBackupDue(scheduled, scheduled.Add(30*time.Second), scheduled.Add(time.Minute)) {
		t.Fatal("must not upload repeatedly within the same window")
	}
	if !remoteBackupDue(next, scheduled.Add(30*time.Second), next) {
		t.Fatal("must run again on the next day")
	}
}

func TestRemoteBackupWindow(t *testing.T) {
	scheduled := time.Date(2026, 12, 31, 23, 59, 30, 0, time.UTC)
	for _, tc := range []struct {
		name string
		now  time.Time
		last time.Time
		want bool
	}{
		{"before", scheduled.Add(-time.Second), time.Time{}, false},
		{"at", scheduled, time.Time{}, true},
		{"midnight", scheduled.Add(time.Minute), time.Time{}, true},
		{"midnight already attempted", scheduled.Add(time.Minute), scheduled, false},
		{"end of window", scheduled.Add(time.Hour), time.Time{}, true},
		{"expired", scheduled.Add(time.Hour + time.Second), time.Time{}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			date := tc.now.UTC()
			today := time.Date(date.Year(), date.Month(), date.Day(), 23, 59, 30, 0, time.UTC)
			if got := remoteBackupDue(today, tc.last, tc.now); got != tc.want {
				t.Fatalf("due = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestLatestDailyBackup(t *testing.T) {
	root := t.TempDir()
	daily := filepath.Join(root, "zod", "daily")
	if err := os.MkdirAll(filepath.Join(daily, "999"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"9", "10", "100", "invalid", "100.tmp"} {
		if err := os.WriteFile(filepath.Join(daily, name), []byte("backup"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "zod", "200"), []byte("outside daily"), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := latestDailyBackup(root, "zod")
	if err != nil || got != filepath.Join(daily, "100") {
		t.Fatalf("latest backup = %q, %v", got, err)
	}
	got, err = latestDailyBackup(root, "nec")
	if err != nil || got != "" {
		t.Fatalf("missing daily directory = %q, %v", got, err)
	}
}
