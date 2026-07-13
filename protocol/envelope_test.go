package protocol

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestResponseRoundTrip(t *testing.T) {
	resp := Fail(
		NewError(ElementNotFound, "missing submit button", map[string]any{"selector": "css=button[type=submit]"}),
		[]Artifact{{Type: "screenshot", Path: "/tmp/failure.png"}},
		Meta{Profile: "work", Tab: "tab_01", DurationMs: 123},
	)

	encoded, err := Encode(resp)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	decoded, err := DecodeStrict[Response](strings.NewReader(string(encoded)))
	if err != nil {
		t.Fatalf("DecodeStrict() error = %v", err)
	}

	if decoded.OK {
		t.Fatalf("decoded.OK = true, want false")
	}
	if decoded.APIVersion != APIVersion {
		t.Fatalf("api_version = %d, want %d", decoded.APIVersion, APIVersion)
	}
	if decoded.Error == nil || decoded.Error.Code != ElementNotFound {
		t.Fatalf("error = %#v, want code %s", decoded.Error, ElementNotFound)
	}
	if len(decoded.Artifacts) != 1 || decoded.Artifacts[0].Path != "/tmp/failure.png" {
		t.Fatalf("artifacts = %#v", decoded.Artifacts)
	}
	if decoded.Meta == nil {
		t.Fatal("meta = nil, want populated meta")
	}
	if decoded.Meta.Profile != "work" || decoded.Meta.Tab != "tab_01" || decoded.Meta.DurationMs != 123 {
		t.Fatalf("meta = %#v", decoded.Meta)
	}
}

func TestRequestRoundTripRawArgs(t *testing.T) {
	req := Request{
		APIVersion: APIVersion,
		Cmd:        "page.goto",
		Args:       json.RawMessage(`{"url":"https://example.com"}`),
		Profile:    "work",
		Tab:        "active",
		TimeoutMs:  15000,
	}

	encoded, err := Encode(req)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	decoded, err := DecodeStrict[Request](strings.NewReader(string(encoded)))
	if err != nil {
		t.Fatalf("DecodeStrict() error = %v", err)
	}

	if decoded.Cmd != req.Cmd || decoded.Profile != req.Profile || decoded.Tab != req.Tab || decoded.TimeoutMs != req.TimeoutMs {
		t.Fatalf("decoded = %#v, want %#v", decoded, req)
	}
	if string(decoded.Args) != string(req.Args) {
		t.Fatalf("args = %s, want %s", decoded.Args, req.Args)
	}
}

func TestDecodeStrictRejectsUnknownFields(t *testing.T) {
	_, err := DecodeStrict[Request](strings.NewReader(`{"api_version":1,"cmd":"ping","surprise":true}`))
	if err == nil {
		t.Fatal("DecodeStrict() error = nil, want INVALID_REQUEST")
	}
	perr, ok := err.(*Error)
	if !ok {
		t.Fatalf("error type = %T, want *Error", err)
	}
	if perr.Code != InvalidRequest {
		t.Fatalf("code = %s, want %s", perr.Code, InvalidRequest)
	}
}

func TestDecodeStrictRejectsMultipleJSONValues(t *testing.T) {
	_, err := DecodeStrict[Request](strings.NewReader(`{"api_version":1,"cmd":"ping"} {}`))
	if err == nil {
		t.Fatal("DecodeStrict() error = nil, want INVALID_REQUEST")
	}
	if got := err.(*Error).Code; got != InvalidRequest {
		t.Fatalf("code = %s, want %s", got, InvalidRequest)
	}
}
