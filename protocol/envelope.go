package protocol

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

const APIVersion = 1

type Request struct {
	APIVersion int             `json:"api_version"`
	Cmd        string          `json:"cmd"`
	Args       json.RawMessage `json:"args,omitempty"`
	Profile    string          `json:"profile,omitempty"`
	Tab        string          `json:"tab,omitempty"`
	TimeoutMs  int64           `json:"timeout_ms,omitempty"`
}

type Response struct {
	OK         bool       `json:"ok"`
	APIVersion int        `json:"api_version"`
	Data       any        `json:"data,omitempty"`
	Error      *Error     `json:"error,omitempty"`
	Artifacts  []Artifact `json:"artifacts,omitempty"`
	Meta       *Meta      `json:"meta,omitempty"`
}

type Artifact struct {
	Type string `json:"type"`
	Path string `json:"path"`
}

type Meta struct {
	Profile    string `json:"profile,omitempty"`
	Tab        string `json:"tab,omitempty"`
	DurationMs int64  `json:"duration_ms,omitempty"`
}

func OK(data any, meta Meta) Response {
	return Response{OK: true, APIVersion: APIVersion, Data: data, Meta: metaPtr(meta)}
}

func Fail(err *Error, artifacts []Artifact, meta Meta) Response {
	if err == nil {
		err = NewError(InternalError, "internal error", nil)
	}
	return Response{OK: false, APIVersion: APIVersion, Error: err, Artifacts: artifacts, Meta: metaPtr(meta)}
}

func metaPtr(meta Meta) *Meta {
	if meta.Profile == "" && meta.Tab == "" && meta.DurationMs == 0 {
		return nil
	}
	return &meta
}

func DecodeStrict[T any](r io.Reader) (T, error) {
	var out T
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&out); err != nil {
		return out, NewError(InvalidRequest, fmt.Sprintf("invalid JSON: %v", err), nil)
	}
	if dec.Decode(&struct{}{}) != io.EOF {
		return out, NewError(InvalidRequest, "invalid JSON: multiple JSON values", nil)
	}
	return out, nil
}

func Encode(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}
