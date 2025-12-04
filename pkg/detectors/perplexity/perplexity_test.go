package perplexity

import (
	"context"
	"testing"

	"github.com/trufflesecurity/trufflehog/v3/pkg/detectors"
	"github.com/trufflesecurity/trufflehog/v3/pkg/pb/detectorspb"
)

func TestPerplexity_Pattern(t *testing.T) {
	tests := []struct {
		name  string
		data  string
		want  int
		match bool
	}{
		{
			name:  "valid perplexity key",
			data:  "perplexity_key = pplx-1234567890abcdef1234567890abcdef",
			want:  1,
			match: true,
		},
		{
			name:  "invalid pattern",
			data:  "perplexity_key = invalid-key",
			want:  0,
			match: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Scanner{}
			results, err := s.FromData(context.Background(), false, []byte(tt.data))
			if err != nil {
				t.Errorf("Perplexity.FromData() error = %v", err)
				return
			}

			if len(results) != tt.want {
				t.Errorf("Perplexity.FromData() got %v results, want %v", len(results), tt.want)
				return
			}

			if tt.match && len(results) > 0 {
				if results[0].DetectorType != detectorspb.DetectorType_Perplexity {
					t.Errorf("Perplexity.FromData() got detector type %v, want %v",
						results[0].DetectorType, detectorspb.DetectorType_Perplexity)
				}
			}
		})
	}
}

func TestPerplexity_Keywords(t *testing.T) {
	s := Scanner{}
	keywords := s.Keywords()
	if len(keywords) == 0 {
		t.Error("Perplexity.Keywords() returned empty slice")
	}
}
