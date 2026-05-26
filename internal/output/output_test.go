package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestWriteJSON_NoPendingNotice(t *testing.T) {
	orig := PendingNotice
	PendingNotice = nil
	defer func() { PendingNotice = orig }()

	var buf bytes.Buffer
	err := WriteJSON(&buf, Envelope{OK: true, Data: map[string]any{"key": "val"}})
	if err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := got["_notice"]; ok {
		t.Error("expected no _notice when PendingNotice is nil")
	}
	if got["ok"] != true {
		t.Error("expected ok=true")
	}
}

func TestWriteJSON_WithPendingNotice(t *testing.T) {
	orig := PendingNotice
	PendingNotice = func() any {
		return map[string]any{"skills": map[string]any{"command": "luma-cli update"}}
	}
	defer func() { PendingNotice = orig }()

	var buf bytes.Buffer
	err := WriteJSON(&buf, Envelope{OK: true})
	if err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	notice, ok := got["_notice"].(map[string]any)
	if !ok {
		t.Fatal("expected _notice field")
	}
	skills, ok := notice["skills"].(map[string]any)
	if !ok {
		t.Fatal("expected _notice.skills")
	}
	if skills["command"] != "luma-cli update" {
		t.Errorf("unexpected command: %v", skills["command"])
	}
}

func TestWriteJSON_PendingNoticeReturnsNil(t *testing.T) {
	orig := PendingNotice
	PendingNotice = func() any { return nil }
	defer func() { PendingNotice = orig }()

	var buf bytes.Buffer
	err := WriteJSON(&buf, Envelope{OK: true})
	if err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := got["_notice"]; ok {
		t.Error("expected no _notice when PendingNotice returns nil")
	}
}

func TestWriteJSON_ExplicitNoticeNotOverwritten(t *testing.T) {
	orig := PendingNotice
	PendingNotice = func() any { return map[string]any{"skills": "injected"}}
	defer func() { PendingNotice = orig }()

	explicit := map[string]any{"custom": "value"}
	var buf bytes.Buffer
	err := WriteJSON(&buf, Envelope{OK: true, Notice: explicit})
	if err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	notice := got["_notice"].(map[string]any)
	if notice["custom"] != "value" {
		t.Error("explicit notice was overwritten")
	}
}

func TestWriteJSON_ErrorEnvelope(t *testing.T) {
	orig := PendingNotice
	PendingNotice = nil
	defer func() { PendingNotice = orig }()

	var buf bytes.Buffer
	err := WriteJSON(&buf, Envelope{OK: false, Code: "test_error", Error: "something went wrong"})
	if err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got["ok"] != false {
		t.Error("expected ok=false")
	}
	if got["error"] != "something went wrong" {
		t.Errorf("unexpected error: %v", got["error"])
	}
}

func TestWriteJSON_OutputEndsWithNewline(t *testing.T) {
	orig := PendingNotice
	PendingNotice = nil
	defer func() { PendingNotice = orig }()

	var buf bytes.Buffer
	_ = WriteJSON(&buf, Envelope{OK: true})
	if !strings.HasSuffix(buf.String(), "\n") {
		t.Error("expected output to end with newline")
	}
}

func TestGetNotice_Nil(t *testing.T) {
	orig := PendingNotice
	PendingNotice = nil
	defer func() { PendingNotice = orig }()

	if got := GetNotice(); got != nil {
		t.Error("GetNotice should return nil when PendingNotice is nil")
	}
}

func TestGetNotice_ReturnsValue(t *testing.T) {
	orig := PendingNotice
	PendingNotice = func() any { return map[string]any{"key": "val"} }
	defer func() { PendingNotice = orig }()

	got := GetNotice()
	if got == nil {
		t.Fatal("GetNotice should return non-nil when PendingNotice is set")
	}
	m, ok := got.(map[string]any)
	if !ok || m["key"] != "val" {
		t.Error("GetNotice returned wrong value")
	}
}
