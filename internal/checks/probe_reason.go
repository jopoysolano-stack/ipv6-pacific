package checks

import (
	"context"
	"errors"
	"net"
	"strings"
	"syscall"
)

// Stable probe failure tokens for public JSON/UI (never raw Go errors).
const (
	ProbeOK          = ""
	ProbeTimeout     = "timeout"
	ProbeRefused     = "refused"
	ProbeNoBanner    = "no_banner"
	ProbeHTTP5xx     = "http_5xx"
	ProbeDNSError    = "dns_error"
	ProbeUnreachable = "unreachable"
	ProbeOther       = "error"
)

func classifyNetError(err error) string {
	if err == nil {
		return ProbeOK
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return ProbeTimeout
	}
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return ProbeTimeout
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		if errors.Is(opErr.Err, syscall.ECONNREFUSED) {
			return ProbeRefused
		}
		if errors.Is(opErr.Err, syscall.ENETUNREACH) || errors.Is(opErr.Err, syscall.EHOSTUNREACH) {
			return ProbeUnreachable
		}
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "refused"):
		return ProbeRefused
	case strings.Contains(msg, "timeout") || strings.Contains(msg, "deadline exceeded"):
		return ProbeTimeout
	case strings.Contains(msg, "no route") || strings.Contains(msg, "unreachable"):
		return ProbeUnreachable
	case strings.Contains(msg, "no such host") || strings.Contains(msg, "server misbehaving"):
		return ProbeDNSError
	default:
		return ProbeOther
	}
}
