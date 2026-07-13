package protocol

import (
	"errors"
	"testing"
)

func TestExitCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "nil", err: nil, want: ExitOK},
		{name: "invalid request", err: NewError(InvalidRequest, "", nil), want: ExitUsage},
		{name: "daemon unavailable", err: NewError(DaemonUnavailable, "", nil), want: ExitDaemonUnavailable},
		{name: "domain error", err: NewError(ElementNotFound, "", nil), want: ExitActionDomainError},
		{name: "generic error", err: errors.New("boom"), want: ExitActionDomainError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExitCode(tt.err); got != tt.want {
				t.Fatalf("ExitCode() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestResponseExitCode(t *testing.T) {
	if got := ResponseExitCode(OK(map[string]string{"pong": "true"}, Meta{})); got != ExitOK {
		t.Fatalf("ResponseExitCode(ok) = %d, want %d", got, ExitOK)
	}
	if got := ResponseExitCode(Fail(NewError(InvalidRequest, "", nil), nil, Meta{})); got != ExitUsage {
		t.Fatalf("ResponseExitCode(invalid) = %d, want %d", got, ExitUsage)
	}
}
