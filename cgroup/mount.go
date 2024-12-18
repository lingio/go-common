package cgroup

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

type (
	Version int
	fsType  string
)

const (
	vNone Version = iota
	V1
	V2

	fsNone fsType = ""
	fsV1   fsType = "cgroup"
	fsV2   fsType = "cgroup2"
)

// https://www.man7.org/linux/man-pages/man5/proc_pid_mountinfo.5.html
// 731 771 0:59 /sysrq-trigger /proc/sysrq-trigger ro,nosuid,nodev,noexec,relatime - proc proc rw
//
// 36 35 98:0 /mnt1 /mnt2 rw,noatime master:1 - ext3 /dev/root rw,errors=continue
// (1)(2)(3)   (4)   (5)      (6)      (7)   (8) (9)   (10)         (11)
//
// (1)  mount ID: a unique ID for the mount (may be reused after umount(2)).
// (2)  parent ID: the ID of the parent mount (or of self for the root of this mount namespace's mount tree).
// (3)  major:minor: the value of st_dev for files on this filesystem (see stat(2)).
// (4)  root: the pathname of the directory in the filesystem which forms the root of this mount.
// (5)  mount point: the pathname of the mount point relative to the process's root directory.
// (6)  mount options: per-mount options (see mount(2)).
// (7)  optional fields: zero or more fields of the form "tag[:value]"; see below.
// (8)  separator: the end of the optional fields is marked by a single hyphen.
// (9)  filesystem type: the filesystem type in the form "type[.subtype]".
// (10) mount source: filesystem-specific information or "none".
// (11) super options: per-superblock options (see mount(2)).
type MountInfo struct {
	Root           string
	MountPoint     string
	FilesystemType fsType
	SuperOptions   string
}

// detectVersion detects the cgroup version from the mountinfo.
func detectVersion(mis []MountInfo) (Version, bool) {
	var v Version
	for _, mi := range mis {
		switch mi.FilesystemType {
		case fsV1:
			v = V1
		case fsV2:
			v = V2
		}
	}
	return v, v != vNone
}

func parseMountInfo(r io.Reader) ([]MountInfo, error) {
	var (
		s   = bufio.NewScanner(r)
		mis []MountInfo
	)
	for s.Scan() {
		line := s.Text()

		mi, err := mountInfoFromLine(line)
		if err != nil {
			return nil, fmt.Errorf("failed to parse mountinfo file %q: %w", line, err)
		}

		mis = append(mis, mi)
	}
	if err := s.Err(); err != nil {
		return nil, err
	}

	return mis, nil
}

func mountInfoFromLine(line string) (MountInfo, error) {
	if line == "" {
		return MountInfo{}, errors.New("empty line")
	}

	fieldss := strings.SplitN(line, " - ", 2)
	if len(fieldss) != 2 {
		return MountInfo{}, fmt.Errorf("invalid separator")
	}

	fields1 := strings.Split(fieldss[0], " ")
	if len(fields1) < 6 {
		return MountInfo{}, fmt.Errorf("not enough fields before separator: %v", fields1)
	} else if len(fields1) > 7 {
		return MountInfo{}, fmt.Errorf("too many fields before separator: %v", fields1)
	} else if len(fields1) == 6 {
		fields1 = append(fields1, "")
	}

	fields2 := strings.Split(fieldss[1], " ")
	if len(fields2) < 3 {
		return MountInfo{}, fmt.Errorf("not enough fields after separator: %v", fields2)
	} else if len(fields2) > 3 {
		return MountInfo{}, fmt.Errorf("too many fields after separator: %v", fields2)
	}

	return MountInfo{
		Root:           fields1[3],
		MountPoint:     fields1[4],
		FilesystemType: fsType(fields2[0]),
		SuperOptions:   fields2[2],
	}, nil
}
