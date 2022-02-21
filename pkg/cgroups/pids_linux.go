//go:build linux
// +build linux

package cgroups

import (
	"path/filepath"
	"strconv"

	"github.com/opencontainers/runc/libcontainer/cgroups"
	"github.com/opencontainers/runc/libcontainer/configs"
)

type linuxPidHandler struct {
}

func getPidsHandler() *linuxPidHandler {
	return &linuxPidHandler{}
}

// Apply set the specified constraints
func (c *linuxPidHandler) Apply(ctr *CgroupControl, res *configs.Resources) error {
	if ctr.cgroup2 {
		path := filepath.Join(cgroupRoot, ctr.config.Path)
		if val := strconv.FormatInt(res.PidsLimit, 10); val != "" {
			if err := WriteFile(path, "pids.max", val); err != nil {
				return err
			}
		}
		return nil
	}
	// maintaining the fs2 and fs1 functions here for future development
	path := filepath.Join(cgroupRoot, Pids, ctr.config.Path)
	if res.PidsLimit != 0 {
		limit := "max"

		if res.PidsLimit > 0 {
			limit = strconv.FormatInt(res.PidsLimit, 10)
		}

		if err := WriteFile(path, "pids.max", limit); err != nil {
			return err
		}
	}

	return nil
}

// Create the cgroup
func (c *linuxPidHandler) Create(ctr *CgroupControl) (bool, error) {
	if ctr.cgroup2 {
		return false, nil
	}
	return ctr.createCgroupDirectory(Pids)
}

// Destroy the cgroup
func (c *linuxPidHandler) Destroy(ctr *CgroupControl) error {
	return rmDirRecursively(ctr.getCgroupv1Path(Pids))
}

// Stat fills a metrics structure with usage stats for the controller
func (c *linuxPidHandler) Stat(ctr *CgroupControl, m *cgroups.Stats) error {
	if ctr.config.Path == "" {
		// nothing we can do to retrieve the pids.current path
		return nil
	}

	var PIDRoot string
	if ctr.cgroup2 {
		PIDRoot = filepath.Join(cgroupRoot, ctr.config.Path)
	} else {
		PIDRoot = ctr.getCgroupv1Path(Pids)
	}

	current, err := readFileAsUint64(filepath.Join(PIDRoot, "pids.current"))
	if err != nil {
		return err
	}

	m.PidsStats.Current = current
	return nil
}
