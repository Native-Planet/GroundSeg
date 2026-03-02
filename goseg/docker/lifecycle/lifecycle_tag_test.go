package lifecycle

import "testing"

func TestImageTagFromReferenceParsesTagsFromRightSide(t *testing.T) {
	cases := []struct {
		name string
		in   string
		out  string
	}{
		{
			name: "name with explicit tag and digest",
			in:   "repo/image:stable@sha256:abcd",
			out:  "stable",
		},
		{
			name: "namespaced image with port registry",
			in:   "localhost:5000/repo/image:dev@sha256:abcd",
			out:  "dev",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := imageTagFromReference(tc.in)
			if got != tc.out {
				t.Fatalf("expected tag %q for %q, got %q", tc.out, tc.in, got)
			}
		})
	}
}

func TestImageTagFromReferenceFallsBackWithoutTag(t *testing.T) {
	cases := []string{
		"localhost:5000/repo/image",
		"repo/image",
		"repo/image@sha256:abcd",
	}

	for _, input := range cases {
		t.Run(input, func(t *testing.T) {
			if got := imageTagFromReference(input); got != "" {
				t.Fatalf("expected no tag for %q, got %q", input, got)
			}
		})
	}
}
