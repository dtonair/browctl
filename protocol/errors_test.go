package protocol

import "testing"

func TestAllErrorCodesHaveDefaultsAndRetryability(t *testing.T) {
	codes := AllErrorCodes()
	if len(codes) != 18 {
		t.Fatalf("len(AllErrorCodes()) = %d, want 18", len(codes))
	}

	seen := map[ErrorCode]bool{}
	for _, code := range codes {
		if seen[code] {
			t.Fatalf("duplicate error code: %s", code)
		}
		seen[code] = true

		if msg := DefaultMessage(code); msg == "" || msg == "unknown error" {
			t.Fatalf("DefaultMessage(%s) = %q", code, msg)
		}

		err := NewError(code, "", nil)
		if err.Code != code {
			t.Fatalf("NewError(%s).Code = %s", code, err.Code)
		}
		if err.Message == "" || err.Message == "unknown error" {
			t.Fatalf("NewError(%s).Message = %q", code, err.Message)
		}
		if err.Retryable != IsRetryable(code) {
			t.Fatalf("NewError(%s).Retryable = %v, want %v", code, err.Retryable, IsRetryable(code))
		}
	}
}

func TestNewErrorKeepsMessageDetailsAndRetryableDefault(t *testing.T) {
	details := map[string]any{"selector": "css=.missing"}
	err := NewError(ElementNotFound, "custom", details)
	if err.Message != "custom" {
		t.Fatalf("Message = %q, want custom", err.Message)
	}
	if !err.Retryable {
		t.Fatalf("Retryable = false, want true")
	}
	if err.Details["selector"] != "css=.missing" {
		t.Fatalf("Details = %#v", err.Details)
	}
	if got := err.Error(); got != "ELEMENT_NOT_FOUND: custom" {
		t.Fatalf("Error() = %q", got)
	}
}

func TestRetryabilityTable(t *testing.T) {
	retryable := []ErrorCode{ProfileLocked, BrowserStartFailed, BrowserCrashed, DaemonUnavailable, TargetDetached, NavigationFailed, ElementNotFound, ElementNotVisible, ElementNotInteractable, StaleElementReference, ActionTimeout}
	for _, code := range retryable {
		if !IsRetryable(code) {
			t.Fatalf("IsRetryable(%s) = false, want true", code)
		}
	}

	nonRetryable := []ErrorCode{InvalidRequest, ProfileNotFound, BrowserNotFound, TabNotFound, ElementAmbiguous, PolicyDenied, InternalError}
	for _, code := range nonRetryable {
		if IsRetryable(code) {
			t.Fatalf("IsRetryable(%s) = true, want false", code)
		}
	}
}
