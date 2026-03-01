package response

import "testing"

func TestParsePokeResponse(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		resType  string
		input    string
		value    string
		success  bool
		wantErr  bool
		errCheck string
	}{
		{
			name:    "success token present",
			resType: "success",
			input:   "[0 %avow 0 %noun %success]",
			success: true,
		},
		{
			name:    "success token missing",
			resType: "success",
			input:   "[0 %avow 0 %noun %failure]",
			success: false,
			wantErr: false,
			value:   "",
		},
		{
			name:    "code token extracted",
			resType: "code",
			input:   "%avow something %code ]",
			value:   "code",
		},
		{
			name:     "code missing closing bracket",
			resType:  "code",
			input:    "%avow no-termination",
			wantErr:  true,
			errCheck: "+code not in poke response",
			value:    "",
			success:  false,
		},
		{
			name:    "desk not found",
			resType: "desk",
			input:   "[%avow] desk does not yet exist",
			value:   "not-found",
		},
		{
			name:    "desk status parsed",
			resType: "desk",
			input:   "[%avow] app status: active ]",
			value:   "active",
		},
		{
			name:    "desk status malformed",
			resType: "desk",
			input:   "[%avow] app status:",
			value:   "not-found",
		},
		{
			name:     "unknown response type",
			resType:  "unknown",
			input:    "[0 %avow 0 %noun %success]",
			wantErr:  true,
			errCheck: "+code not in poke response",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			value, success, err := ParsePokeResponse(tc.resType, tc.input)
			if value != tc.value {
				t.Fatalf("ParsePokeResponse(%q) value = %q, want %q", tc.resType, value, tc.value)
			}
			if success != tc.success {
				t.Fatalf("ParsePokeResponse(%q) success = %t, want %t", tc.resType, success, tc.success)
			}
			if tc.wantErr && err == nil {
				t.Fatalf("ParsePokeResponse(%q) expected error, got nil", tc.resType)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("ParsePokeResponse(%q) unexpected error: %v", tc.resType, err)
			}
			if tc.errCheck != "" && err != nil && err.Error() != tc.errCheck {
				t.Fatalf("ParsePokeResponse(%q) error = %q, want %q", tc.resType, err.Error(), tc.errCheck)
			}
		})
	}
}
