package storagefacade

import (
	"errors"
	"os"
	"testing"

	"groundseg/system/storage"
)

func TestListHardDisksPropagatesCommandError(t *testing.T) {
	commandErr := errors.New("lsblk failed")
	_, err := ListHardDisks(storage.DiskSeams{
		RunDiskCommandFn: func(string, ...string) (string, error) {
			return "", commandErr
		},
	})
	if err == nil {
		t.Fatal("expected ListHardDisks to return command error")
	}
	if !errors.Is(err, commandErr) {
		t.Fatalf("expected ListHardDisks error to wrap command error, got %v", err)
	}
}

func TestCreateGroundSegFilesystemPropagatesStatError(t *testing.T) {
	statErr := errors.New("stat failed")
	_, err := CreateGroundSegFilesystem("sda", storage.DiskSeams{
		StatFn: func(string) (os.FileInfo, error) {
			return nil, statErr
		},
	})
	if err == nil {
		t.Fatal("expected CreateGroundSegFilesystem to fail when stat fails")
	}
	if !errors.Is(err, statErr) {
		t.Fatalf("expected CreateGroundSegFilesystem error to wrap stat error, got %v", err)
	}
}
