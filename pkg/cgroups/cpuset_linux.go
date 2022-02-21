//go:build linux
// +build linux

package cgroups

import (
	"path/filepath"

	"github.com/opencontainers/runc/libcontainer/cgroups"
	"github.com/opencontainers/runc/libcontainer/configs"
)

type linuxCpusetHandler struct {
}

func getCpusetHandler() *linuxCpusetHandler {
	return &linuxCpusetHandler{}
}

// Apply set the specified constraints
func (c *linuxCpusetHandler) Apply(ctr *CgroupControl, res *configs.Resources) error {
	if ctr.cgroup2 {
		path := filepath.Join(cgroupRoot, ctr.config.Path)
		if res.CpusetCpus != "" {
			if err := WriteFile(path, "cpuset.cpus", res.CpusetCpus); err != nil {
				return err
			}
		}
		if res.CpusetMems != "" {
			if err := WriteFile(path, "cpuset.mems", res.CpusetMems); err != nil {
				return err
			}
		}
		return nil
	}
	// maintaining the fs2 and fs1 functions here for future development
	path := filepath.Join(cgroupRoot, CPUset, ctr.config.Path)
	if res.CpusetCpus != "" {
		if err := WriteFile(path, "cpuset.cpus", res.CpusetCpus); err != nil {
			return err
		}
	}
	if res.CpusetMems != "" {
		if err := WriteFile(path, "cpuset.mems", res.CpusetMems); err != nil {
			return err
		}
	}
	return nil
}

// Create the cgroup
func (c *linuxCpusetHandler) Create(ctr *CgroupControl) (bool, error) {
	if ctr.cgroup2 {
		path := filepath.Join(cgroupRoot, ctr.config.Path)
		return true, cpusetCopyFromParent(path, true)
	}
	created, err := ctr.createCgroupDirectory(CPUset)
	if !created || err != nil {
		return created, err
	}
	return true, cpusetCopyFromParent(ctr.getCgroupv1Path(CPUset), false)
}

// Destroy the cgroup
func (c *linuxCpusetHandler) Destroy(ctr *CgroupControl) error {
	return rmDirRecursively(ctr.getCgroupv1Path(CPUset))
}

// Stat fills a metrics structure with usage stats for the controller
func (c *linuxCpusetHandler) Stat(ctr *CgroupControl, m *cgroups.Stats) error {
	return nil
}
