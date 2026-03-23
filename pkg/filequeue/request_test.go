package filequeue

import (
	"encoding/json"
	"testing"
	"time"
)

func TestRequestMeta_Marshal(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	m := &RequestMeta{
		RequestID:           "a1b2c3d4e5f67890",
		Tool:                "compliance",
		Operation:           "scan",
		Method:              "GET",
		Path:                "/api/v1/compliance/scan",
		Params:              map[string]string{"ruleset": "pci-dss"},
		HasPayload:          false,
		PayloadPath:         "",
		ResponseMetaPath:    "done/a1b2c3d4e5f67890.meta.json",
		ResponsePayloadPath: "payloads/resp/a1b2c3d4e5f67890.json",
		TimeoutSeconds:      120,
		CreatedAt:           now,
		ExpectedResponse:    "json",
	}
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var got RequestMeta
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if got.RequestID != m.RequestID {
		t.Errorf("expected RequestID %q, got %q", m.RequestID, got.RequestID)
	}
	if got.Tool != "compliance" {
		t.Errorf("expected Tool 'compliance', got %q", got.Tool)
	}
	if got.Params["ruleset"] != "pci-dss" {
		t.Errorf("expected ruleset='pci-dss', got %q", got.Params["ruleset"])
	}
	if got.HasPayload {
		t.Error("expected HasPayload=false")
	}
}

func TestRequestMeta_MarshalEmpty(t *testing.T) {
	m := &RequestMeta{RequestID: "a1b2c3d4e5f67890"}
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var got RequestMeta
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if got.HasPayload {
		t.Error("expected HasPayload defaults to false")
	}
}

func TestResponseMeta_MarshalSuccess(t *testing.T) {
	m := &ResponseMeta{
		RequestID:  "a1b2c3d4e5f67890",
		Status:     "success",
		StatusCode: 200,
		HasPayload: true,
		ExitCode:   0,
	}
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var got ResponseMeta
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if got.Status != "success" {
		t.Errorf("expected status='success', got %q", got.Status)
	}
	if got.ExitCode != 0 {
		t.Errorf("expected exit_code=0, got %d", got.ExitCode)
	}
}

func TestResponseMeta_MarshalError(t *testing.T) {
	m := &ResponseMeta{
		RequestID:  "a1b2c3d4e5f67890",
		Status:     "error",
		StatusCode: 503,
		Error:      "API request failed",
		Suggestion: "retry in 30 seconds",
		Retry:      true,
		ExitCode:   1,
	}
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var got ResponseMeta
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if got.Status != "error" {
		t.Errorf("expected status='error', got %q", got.Status)
	}
	if !got.Retry {
		t.Error("expected retry=true")
	}
	if got.Error != "API request failed" {
		t.Errorf("unexpected error field: %q", got.Error)
	}
}
