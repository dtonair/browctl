package protocol

type ErrorCode string

const (
	InvalidRequest         ErrorCode = "INVALID_REQUEST"
	ProfileNotFound        ErrorCode = "PROFILE_NOT_FOUND"
	ProfileLocked          ErrorCode = "PROFILE_LOCKED"
	BrowserNotFound        ErrorCode = "BROWSER_NOT_FOUND"
	BrowserStartFailed     ErrorCode = "BROWSER_START_FAILED"
	BrowserCrashed         ErrorCode = "BROWSER_CRASHED"
	DaemonUnavailable      ErrorCode = "DAEMON_UNAVAILABLE"
	TabNotFound            ErrorCode = "TAB_NOT_FOUND"
	TargetDetached         ErrorCode = "TARGET_DETACHED"
	NavigationFailed       ErrorCode = "NAVIGATION_FAILED"
	ElementNotFound        ErrorCode = "ELEMENT_NOT_FOUND"
	ElementAmbiguous       ErrorCode = "ELEMENT_AMBIGUOUS"
	ElementNotVisible      ErrorCode = "ELEMENT_NOT_VISIBLE"
	ElementNotInteractable ErrorCode = "ELEMENT_NOT_INTERACTABLE"
	StaleElementReference  ErrorCode = "STALE_ELEMENT_REFERENCE"
	ActionTimeout          ErrorCode = "ACTION_TIMEOUT"
	PolicyDenied           ErrorCode = "POLICY_DENIED"
	InternalError          ErrorCode = "INTERNAL_ERROR"
)

var allErrorCodes = []ErrorCode{
	InvalidRequest,
	ProfileNotFound,
	ProfileLocked,
	BrowserNotFound,
	BrowserStartFailed,
	BrowserCrashed,
	DaemonUnavailable,
	TabNotFound,
	TargetDetached,
	NavigationFailed,
	ElementNotFound,
	ElementAmbiguous,
	ElementNotVisible,
	ElementNotInteractable,
	StaleElementReference,
	ActionTimeout,
	PolicyDenied,
	InternalError,
}

type Error struct {
	Code      ErrorCode      `json:"code"`
	Message   string         `json:"message"`
	Retryable bool           `json:"retryable"`
	Details   map[string]any `json:"details,omitempty"`
}

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Message == "" {
		return string(e.Code)
	}
	return string(e.Code) + ": " + e.Message
}

func NewError(code ErrorCode, message string, details map[string]any) *Error {
	if message == "" {
		message = DefaultMessage(code)
	}
	return &Error{Code: code, Message: message, Retryable: IsRetryable(code), Details: details}
}

func AllErrorCodes() []ErrorCode {
	out := make([]ErrorCode, len(allErrorCodes))
	copy(out, allErrorCodes)
	return out
}

func DefaultMessage(code ErrorCode) string {
	switch code {
	case InvalidRequest:
		return "invalid request"
	case ProfileNotFound:
		return "profile not found"
	case ProfileLocked:
		return "profile is locked"
	case BrowserNotFound:
		return "browser not found"
	case BrowserStartFailed:
		return "browser failed to start"
	case BrowserCrashed:
		return "browser crashed"
	case DaemonUnavailable:
		return "daemon unavailable"
	case TabNotFound:
		return "tab not found"
	case TargetDetached:
		return "target detached"
	case NavigationFailed:
		return "navigation failed"
	case ElementNotFound:
		return "element not found"
	case ElementAmbiguous:
		return "element selector matched multiple elements"
	case ElementNotVisible:
		return "element not visible"
	case ElementNotInteractable:
		return "element not interactable"
	case StaleElementReference:
		return "stale element reference"
	case ActionTimeout:
		return "action timed out"
	case PolicyDenied:
		return "policy denied"
	case InternalError:
		return "internal error"
	default:
		return "unknown error"
	}
}

func IsRetryable(code ErrorCode) bool {
	switch code {
	case ProfileLocked,
		BrowserStartFailed,
		BrowserCrashed,
		DaemonUnavailable,
		TargetDetached,
		NavigationFailed,
		ElementNotFound,
		ElementNotVisible,
		ElementNotInteractable,
		StaleElementReference,
		ActionTimeout:
		return true
	default:
		return false
	}
}
