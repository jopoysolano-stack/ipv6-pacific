package checks

import (
	"context"
	"net"
	"syscall"
	"testing"
)

func TestClassifyNetError(t *testing.T) {
	if got := classifyNetError(nil); got != ProbeOK {
		t.Fatalf("nil: got %q", got)
	}
	if got := classifyNetError(context.DeadlineExceeded); got != ProbeTimeout {
		t.Fatalf("deadline: got %q", got)
	}
	op := &net.OpError{Op: "dial", Err: syscall.ECONNREFUSED}
	if got := classifyNetError(op); got != ProbeRefused {
		t.Fatalf("refused: got %q", got)
	}
}
