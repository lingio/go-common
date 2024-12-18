package cgroup

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

var (
	errNoCgroup = errors.New("process is not in cgroup")
)

type (
	// Tree represents the cgroup.
	Tree struct {
		version Version
		groups  []Hierarchy
		mounts  []MountInfo
	}

	ErrNotFound struct {
		wrapped error
		path    string
	}
)

// Autodetect attempts to parse
func Autodetect() (*Tree, error) {
	mf, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		return nil, fmt.Errorf("failed to open /proc/self/mountinfo: %w", err)
	}
	defer mf.Close()

	mis, err := parseMountInfo(mf)
	if err != nil {
		return nil, fmt.Errorf("failed to parse mountinfo: %w", err)
	}

	v, ok := detectVersion(mis)
	if !ok {
		return nil, errNoCgroup
	}

	cf, err := os.Open("/proc/self/cgroup")
	if err != nil {
		return nil, fmt.Errorf("failed to open /proc/self/cgroup: %w", err)
	}
	defer cf.Close()

	chs, err := parseCgroup(cf)
	if err != nil {
		return nil, fmt.Errorf("failed to parse cgroup file: %w", err)
	}

	return &Tree{v, chs, mis}, nil
}

// Field returns the content of the controller interface file iface. Fore a
// comprehensive list of controllers and interface files see:
// https://www.kernel.org/doc/html/latest/admin-guide/cgroup-v2.html#controllers
func (c Tree) InterfaceFile(iface string) (string, error) {
	var (
		nodeIdx  int
		mountIdx int
	)

	if c.version == vNone {
		return "", ErrNotFound{errors.New("unknown cgroup version"), iface}
	} else if c.version == V1 {
		return "", ErrNotFound{errors.New("cgroup v1 interface file is not yet implemented"), iface}
	} else if c.version == V2 {
		nodeIdx = slices.IndexFunc(c.groups, func(ch Hierarchy) bool {
			return ch.HierarchyID == "0" && ch.ControllerList == "" // v2
		})
		mountIdx = slices.IndexFunc(c.mounts, func(mi MountInfo) bool {
			return mi.FilesystemType == fsV2
		})
	}

	//
	if mountIdx == -1 {
		return "", ErrNotFound{errors.New("mountpoint not found"), iface}
	}
	if nodeIdx == -1 {
		return "", ErrNotFound{errors.New("controller not found"), iface}
	}

	var (
		h  = c.groups[nodeIdx]
		mi = c.mounts[mountIdx]
	)

	// resolve the actual cgroup path
	cgroupPath, err := resolveCgroupPath(mi.MountPoint, mi.Root, h.CgroupPath)
	if err != nil {
		return "", ErrNotFound{err, iface}
	}

	// retrieve the memory limit from the memory.max file
	path := filepath.Join(cgroupPath, iface)

	b, err := os.ReadFile(path)
	if err != nil {
		return "", ErrNotFound{err, iface}
	}

	sval := strings.TrimSpace(string(b))
	return sval, nil
}

// resolveCgroupPath resolves the actual cgroup path from the mountpoint, root, and cgroupRelPath.
func resolveCgroupPath(mountpoint, root, cgroupRelPath string) (string, error) {
	rel, err := filepath.Rel(root, cgroupRelPath)
	if err != nil {
		return "", err
	}

	// if the relative path is ".", then the cgroupRelPath is the root itself.
	if rel == "." {
		return mountpoint, nil
	}

	// if the relative path starts with "..", then it is outside the root.
	if strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("invalid cgroup path: %s is not under root %s", cgroupRelPath, root)
	}

	return filepath.Join(mountpoint, rel), nil
}

func (err ErrNotFound) Error() string {
	return fmt.Sprintf("cgroup interface file: %s: %v", err.path, err.wrapped)
}
func (err ErrNotFound) Unwrap() error {
	return err.wrapped
}

type (
	memoryController struct{} // memory.{current,min,low,high,max,reclaim,peak,oom.group,events,envets.local,stat,...}
	cpuController    struct{} // cpu.{stat,weight,weight.nice,max,max.burst,pressure,uclamp.{min,max},idle}
)
