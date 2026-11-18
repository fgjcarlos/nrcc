package model

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestHostStatus_RecommendationCodesJSON(t *testing.T) {
	withCodes, err := json.Marshal(HostStatus{
		Recommendations:     []string{"legacy text"},
		RecommendationCodes: []string{RecommendationPortlessOptional},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"recommendations":["legacy text"]`, `"recommendationCodes":["portless_optional"]`} {
		if !strings.Contains(string(withCodes), want) {
			t.Fatalf("JSON %s missing %s", withCodes, want)
		}
	}

	empty, err := json.Marshal(HostStatus{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(empty), "recommendation") {
		t.Fatalf("empty recommendations must be omitted like before, got %s", empty)
	}
}
