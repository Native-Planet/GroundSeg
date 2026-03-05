package disktest

import (
	"testing"

	"groundseg/structs"
)

type ListHardDisksCase struct {
	Name     string
	Prepare  func()
	ListFunc func() (structs.LSBLKDevice, error)
	Assert   func(t *testing.T, got structs.LSBLKDevice, err error)
}

func RunListHardDisksContractMatrix(t *testing.T, cases []ListHardDisksCase) {
	t.Helper()
	for _, tc := range cases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Helper()
			if tc.Prepare != nil {
				tc.Prepare()
			}
			got, err := tc.ListFunc()
			tc.Assert(t, got, err)
		})
	}
}

type IsMountedMMCCase struct {
	Name       string
	Prepare    func()
	Path       string
	MountedFn  func(string) (bool, error)
	ExpectErr  bool
	ExpectBool bool
}

func RunMountedMMCMatrix(t *testing.T, cases []IsMountedMMCCase) {
	t.Helper()
	for _, tc := range cases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Helper()
			if tc.Prepare != nil {
				tc.Prepare()
			}
			mounted, err := tc.MountedFn(tc.Path)
			if tc.ExpectErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if mounted != tc.ExpectBool {
				t.Fatalf("expected mounted=%v, got %v", tc.ExpectBool, mounted)
			}
		})
	}
}

type ParseBoolLineCase struct {
	Name      string
	Input     string
	ParseFn   func(string) bool
	ShouldHit bool
}

func RunFstabLineParseMatrix(t *testing.T, cases []ParseBoolLineCase) {
	t.Helper()
	for _, tc := range cases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			if got := tc.ParseFn(tc.Input); got != tc.ShouldHit {
				t.Fatalf("expected parse result for %q to be %v, got %v", tc.Input, tc.ShouldHit, got)
			}
		})
	}
}

type ReconcileFstabCase struct {
	Name   string
	Run    func() ([]string, bool)
	Assert func(t *testing.T, reconciled []string, changed bool)
}

func RunReconcileFstabMatrix(t *testing.T, cases []ReconcileFstabCase) {
	t.Helper()
	for _, tc := range cases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			reconciled, changed := tc.Run()
			tc.Assert(t, reconciled, changed)
		})
	}
}
