package protocol

const (
	ExitOK                = 0
	ExitActionDomainError = 1
	ExitUsage             = 2
	ExitDaemonUnavailable = 3
)

func ExitCode(err error) int {
	if err == nil {
		return ExitOK
	}

	perr, ok := err.(*Error)
	if !ok {
		return ExitActionDomainError
	}

	switch perr.Code {
	case InvalidRequest:
		return ExitUsage
	case DaemonUnavailable:
		return ExitDaemonUnavailable
	default:
		return ExitActionDomainError
	}
}

func ResponseExitCode(resp Response) int {
	if resp.OK {
		return ExitOK
	}
	return ExitCode(resp.Error)
}
