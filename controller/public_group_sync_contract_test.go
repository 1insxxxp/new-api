package controller

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestPublicGroupSyncRequestJSONRoundTrip(t *testing.T) {
	input := 0.003
	output := 0.015
	want := PublicGroupSyncRequest{
		Version: PublicGroupSyncSnapshotVersion, EventID: "evt-public-group-001", Revision: 7,
		GroupID: 42, GroupName: "premium", PublicEnabled: true, GroupRatio: 1.25,
		BillingMode: "token", Models: []string{"claude-sonnet"},
		ModelMapping: map[string]string{"claude-sonnet": "claude-sonnet-4-20250514"},
		ModelPricing: map[string]PublicGroupSyncModel{"claude-sonnet": {
			Platform: "anthropic", DisplayName: "claude-sonnet", UpstreamModel: "claude-sonnet-4-20250514",
			BillingMode: "token", InputPrice: &input, OutputPrice: &output,
		}},
	}
	payload, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	var got PublicGroupSyncRequest
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("round-trip mismatch: got %#v, want %#v", got, want)
	}
}
