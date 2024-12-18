package cgroup

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// https://www.man7.org/linux/man-pages/man7/cgroups.7.html
//
// 5:cpuacct,cpu,cpuset:/daemons
// (1)       (2)           (3)
//
// (1) hierarchy ID:
//
//	cgroups version 1 hierarchies, this field
//	contains a unique hierarchy ID number that can be
//	matched to a hierarchy ID in /proc/cgroups.  For the
//	cgroups version 2 hierarchy, this field contains the
//	value 0.
//
// (2) controller list:
//
//	For cgroups version 1 hierarchies, this field
//	contains a comma-separated list of the controllers
//	bound to the hierarchy.  For the cgroups version 2
//	hierarchy, this field is empty.
//
// (3) cgroup path:
//
//	This field contains the pathname of the control group
//	in the hierarchy to which the process belongs.  This
//	pathname is relative to the mount point of the
//	hierarchy.
type Hierarchy struct {
	HierarchyID    string
	ControllerList string
	CgroupPath     string
}

func parseCgroup(r io.Reader) ([]Hierarchy, error) {
	var (
		s   = bufio.NewScanner(r)
		chs []Hierarchy
	)
	for s.Scan() {
		line := s.Text()

		ch, err := hierarchyFromLine(line)
		if err != nil {
			return nil, fmt.Errorf("failed to parse cgroup file %q: %w", line, err)
		}

		chs = append(chs, ch)
	}
	if err := s.Err(); err != nil {
		return nil, err
	}

	return chs, nil
}

func hierarchyFromLine(line string) (Hierarchy, error) {
	fields := strings.Split(line, ":")
	if len(fields) < 3 {
		return Hierarchy{}, fmt.Errorf("not enough fields: %v", fields)
	} else if len(fields) > 3 {
		return Hierarchy{}, fmt.Errorf("too many fields: %v", fields)
	}

	return Hierarchy{
		HierarchyID:    fields[0],
		ControllerList: fields[1],
		CgroupPath:     fields[2],
	}, nil
}
