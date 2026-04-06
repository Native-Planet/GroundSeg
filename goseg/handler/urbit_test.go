package handler

import "testing"

func TestLooksLikeObjectStoreAlias(t *testing.T) {
	tests := []struct {
		alias string
		want  bool
	}{
		{alias: "s3.sovsef-risfex-sitful-hatred.nativeplanet.live", want: true},
		{alias: "bucket.s3.sovsef-risfex-sitful-hatred.nativeplanet.live", want: true},
		{alias: "console.s3.sovsef-risfex-sitful-hatred.nativeplanet.live", want: true},
		{alias: "sovsef-risfex-sitful-hatred.nativeplanet.live", want: false},
		{alias: "ship.example.com", want: false},
	}

	for _, test := range tests {
		got := looksLikeObjectStoreAlias("sovsef-risfex-sitful-hatred", test.alias)
		if got != test.want {
			t.Fatalf("looksLikeObjectStoreAlias(%q) = %v, want %v", test.alias, got, test.want)
		}
	}
}
