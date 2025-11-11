package netutil

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"syscall"
)

// GetTimeoutInfo inspects an error and determines if it represents
// a timeout, returning (true, description) if it does.
func GetTimeoutInfo(err error) (bool, string) {
	if err == nil {
		return false, ""
	}

	// 1️⃣ Context deadline exceeded
	if errors.Is(err, context.DeadlineExceeded) {
		return true, "context deadline exceeded"
	}

	// 2️⃣ net.Error interface (Timeout or Temporary)
	var nerr net.Error
	if errors.As(err, &nerr) {
		if nerr.Timeout() {
			// Try to add detail from net.OpError if available
			var opErr *net.OpError
			if errors.As(err, &opErr) {
				if opErr.Op != "" {
					return true, fmt.Sprintf("%s timeout", opErr.Op)
				}
			}
			return true, "network timeout"
		}
		if nerr.Temporary() {
			return false, "temporary network error"
		}
	}

	// 3️⃣ url.Error wrapping a net.Error
	var uerr *url.Error
	if errors.As(err, &uerr) {
		if ne, ok := uerr.Err.(net.Error); ok && ne.Timeout() {
			if uerr.Op != "" {
				return true, fmt.Sprintf("%s timeout (url.Error)", uerr.Op)
			}
			return true, "timeout (url.Error)"
		}
	}

	// 4️⃣ syscall.Errno directly
	var errno syscall.Errno
	if errors.As(err, &errno) {
		switch errno {
		case syscall.ETIMEDOUT:
			return true, "syscall ETIMEDOUT"
		case syscall.ETIME:
			return true, "syscall ETIME"
		}
	}

	// 5️⃣ os.SyscallError wrapper
	var sysErr *os.SyscallError
	if errors.As(err, &sysErr) {
		switch sysErr.Err {
		case syscall.ETIMEDOUT:
			return true, fmt.Sprintf("syscall %s ETIMEDOUT", sysErr.Syscall)
		case syscall.ETIME:
			return true, fmt.Sprintf("syscall %s ETIME", sysErr.Syscall)
		default:
			return false, fmt.Sprintf("syscall %s: %v", sysErr.Syscall, sysErr.Err)
		}
	}

	// 6️⃣ net.OpError that isn't Timeout() but still a connect issue
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		switch opErr.Op {
		case "dial", "connect":
			return true, fmt.Sprintf("%s failure: %v", opErr.Op, opErr.Err)
		}
	}

	// 7️⃣ Fallback
	return false, ""
}
