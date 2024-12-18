package common

import (
	"errors"
	"fmt"
	"os"
	"runtime/debug"
	"strconv"

	"github.com/lingio/go-common/cgroup"
)

// SetMemoryLimitFromCgroup autodetects current cgroup memory.max and provides
// the go runtime with a soft memory limit (GOMEMLIMIT) computed as f times
// the cgroup value. Recommended value for f is 0.9 when memory.max > 100MB.
//
// When successful, returns the new limit and nil error. Otherwise returns the
// current limit and a non-nil error.
func SetMemoryLimitFromCgroup(f float64) (int64, error) {
	curlimit := debug.SetMemoryLimit(-1)

	if _, ok := os.LookupEnv("GOMEMLIMIT"); ok {
		return curlimit, errors.New("GOMEMLIMIT env is set")
	}

	grp, err := cgroup.Autodetect()
	if err != nil {
		return curlimit, err
	}

	slimit, err := grp.InterfaceFile("memory.max")
	if err != nil {
		return curlimit, err
	}

	if slimit == "max" {
		return curlimit, errors.New("memory.max = max")
	}

	limit, err := strconv.ParseUint(slimit, 10, 64)
	if err != nil {
		return curlimit, fmt.Errorf("failed to parse memory.max value: %w", err)
	}

	if limit == 0 {
		return curlimit, errors.New("memory.max = 0")
	}

	newlimit := int64(float64(limit) * 0.9)
	if newlimit < 1 {
		return curlimit, fmt.Errorf("new memory limit %d < 1", newlimit)
	}

	debug.SetMemoryLimit(newlimit)
	return newlimit, nil
}
