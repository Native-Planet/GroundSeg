package disktest

import (
	"testing"

	"groundseg/structs"
)

func TestRunListHardDisksContractMatrixRunsEachCase(t *testing.T) {
	t.Parallel()

	calls := 0
	RunListHardDisksContractMatrix(t, []ListHardDisksCase{
		{
			Name: "success",
			ListFunc: func() (structs.LSBLKDevice, error) {
				calls++
				return structs.LSBLKDevice{
					BlockDevices: []structs.BlockDev{{Name: "sda"}},
				}, nil
			},
			Assert: func(t *testing.T, got structs.LSBLKDevice, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(got.BlockDevices) != 1 || got.BlockDevices[0].Name != "sda" {
					t.Fatalf("unexpected devices: %+v", got.BlockDevices)
				}
			},
		},
		{
			Name: "prepare hook",
			Prepare: func() {
				calls++
			},
			ListFunc: func() (structs.LSBLKDevice, error) {
				return structs.LSBLKDevice{}, nil
			},
			Assert: func(*testing.T, structs.LSBLKDevice, error) {},
		},
	})

	if calls != 2 {
		t.Fatalf("expected two matrix callback invocations, got %d", calls)
	}
}

func TestRunMountedMMCMatrixHandlesSuccessAndExpectedError(t *testing.T) {
	t.Parallel()

	RunMountedMMCMatrix(t, []IsMountedMMCCase{
		{
			Name:       "mounted true",
			Path:       "/tmp",
			MountedFn:  func(string) (bool, error) { return true, nil },
			ExpectBool: true,
		},
		{
			Name:      "expected error",
			Path:      "/tmp",
			MountedFn: func(string) (bool, error) { return false, assertErr{} },
			ExpectErr: true,
		},
	})
}

func TestRunFstabLineParseMatrixChecksExpectedResult(t *testing.T) {
	t.Parallel()

	RunFstabLineParseMatrix(t, []ParseBoolLineCase{
		{
			Name:      "parses",
			Input:     "entry",
			ParseFn:   func(string) bool { return true },
			ShouldHit: true,
		},
		{
			Name:      "rejects",
			Input:     "comment",
			ParseFn:   func(string) bool { return false },
			ShouldHit: false,
		},
	})
}

func TestRunReconcileFstabMatrixRunsAssertions(t *testing.T) {
	t.Parallel()

	RunReconcileFstabMatrix(t, []ReconcileFstabCase{
		{
			Name: "changed",
			Run: func() ([]string, bool) {
				return []string{"line"}, true
			},
			Assert: func(t *testing.T, reconciled []string, changed bool) {
				if !changed {
					t.Fatal("expected changed=true")
				}
				if len(reconciled) != 1 {
					t.Fatalf("unexpected reconciled lines: %+v", reconciled)
				}
			},
		},
	})
}

type assertErr struct{}

func (assertErr) Error() string { return "assert error" }
