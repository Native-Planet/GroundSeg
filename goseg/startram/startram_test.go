package startram

import (
	"reflect"
	"testing"

	"groundseg/structs"
)

func TestPendingSubdomains(t *testing.T) {
	tests := []struct {
		name         string
		subdomains   []structs.Subdomain
		expectedURLs []string
		want         []string
	}{
		{
			name: "all expected services are ready",
			subdomains: []structs.Subdomain{
				{URL: "sampel-palnet", Status: "ok"},
				{URL: "s3.sampel-palnet", Status: "ok"},
			},
			expectedURLs: []string{"sampel-palnet", "s3.sampel-palnet"},
			want:         []string{},
		},
		{
			name: "later service is still creating",
			subdomains: []structs.Subdomain{
				{URL: "sampel-palnet", Status: "ok"},
				{URL: "s3.sampel-palnet", Status: "creating"},
			},
			expectedURLs: []string{"sampel-palnet", "s3.sampel-palnet"},
			want:         []string{"s3.sampel-palnet creating"},
		},
		{
			name: "expected service missing",
			subdomains: []structs.Subdomain{
				{URL: "sampel-palnet", Status: "ok"},
			},
			expectedURLs: []string{"sampel-palnet", "s3.sampel-palnet"},
			want:         []string{"s3.sampel-palnet missing"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pendingSubdomains(tt.subdomains, tt.expectedURLs)
			if got == nil {
				got = []string{}
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("pendingSubdomains() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
