//go:build linux
// +build linux

package cgroups

import (
	"path/filepath"
	"strconv"

	"github.com/opencontainers/runc/libcontainer/cgroups"
	"github.com/opencontainers/runc/libcontainer/cgroups/fs"
	"github.com/opencontainers/runc/libcontainer/configs"
)

type linuxMemHandler struct {
	Mem fs.MemoryGroup
}

func getMemoryHandler() *linuxMemHandler {
	return &linuxMemHandler{}
}

// Apply set the specified constraints
func (c *linuxMemHandler) Apply(ctr *CgroupControl, res *configs.Resources) error {
	if ctr.cgroup2 {
		path := filepath.Join(cgroupRoot, ctr.config.Path)
		swap, err := cgroups.ConvertMemorySwapToCgroupV2Value(res.MemorySwap, res.Memory)
		if err != nil {
			return err
		}
		swapStr := strconv.FormatInt(swap, 10)
		if swapStr == "" && swap == 0 && res.MemorySwap > 0 {
			swapStr = "0"
		}
		if swapStr != "" {
			if err := WriteFile(path, "memory.swap.max", swapStr); err != nil {
				return err
			}
		}

		if val := strconv.FormatInt(res.Memory, 10); val != "" {
			if err := WriteFile(path, "memory.max", val); err != nil {
				return err
			}
		}

		if val := strconv.FormatInt(res.MemoryReservation, 10); val != "" {
			if err := WriteFile(path, "memory.low", val); err != nil {
				return err
			}
		}
		return nil
	}
	// maintaining the fs2 and fs1 functions here for future development
	path := filepath.Join(cgroupRoot, Memory, ctr.config.Path)
	/*
		if res.Memory == -1 && res.MemorySwap == 0 {
			// Only set swap if it's enabled in kernel
			if _, err := os.Open(filepath.Join(path, "memory.memsw.limit_in_bytes")); err == nil {
				res.MemorySwap = -1
			}
		}

		if res.Memory != 0 && res.MemorySwap != 0 {

			limitString, err := ReadFile(path, "memory.limit_in_bytes")
			if err != nil {
				return err
			}

			limitString = strings.TrimSpace(limitString)

			curLimit, err := strconv.ParseUint(limitString, 10, 64)
			if err != nil {
				return err
			}

			// When update memory limit, we should adapt the write sequence
			// for memory and swap memory, so it won't fail because the new
			// value and the old value don't fit kernel's validation.
			if res.MemorySwap == -1 || curLimit < uint64(res.MemorySwap) {
				if err := cgroups.WriteFile(path, "memory.memsw.limit_in_bytes", strconv.FormatInt(res.MemorySwap, 10)); err != nil {
					return err
				}
				if err := cgroups.WriteFile(path, "memory.limit_in_bytes", strconv.FormatInt(res.Memory, 10)); err != nil {
					return err
				}
				return nil
			}
		}

		if res.MemoryReservation != 0 {
			if err := cgroups.WriteFile(path, "memory.soft_limit_in_bytes", strconv.FormatInt(res.MemoryReservation, 10)); err != nil {
				return err
			}
		}

		if res.OomKillDisable {
			if err := cgroups.WriteFile(path, "memory.oom_control", "1"); err != nil {
				return err
			}
		}

		switch {
		case res.MemorySwappiness == nil || int64(*res.MemorySwappiness) == -1:
			return nil
		case *res.MemorySwappiness <= 100:
			if err := cgroups.WriteFile(path, "memory.swappiness", strconv.FormatUint(*res.MemorySwappiness, 10)); err != nil {
				return err
			}
		default:
			return fmt.Errorf("invalid memory swappiness value: %d (valid range is 0-100)", res.MemorySwappiness)
		}
		return nil
	*/
	return c.Mem.Set(path, res)
}

// Create the cgroup
func (c *linuxMemHandler) Create(ctr *CgroupControl) (bool, error) {
	if ctr.cgroup2 {
		return false, nil
	}
	return ctr.createCgroupDirectory(Memory)
}

// Destroy the cgroup
func (c *linuxMemHandler) Destroy(ctr *CgroupControl) error {
	return rmDirRecursively(ctr.getCgroupv1Path(Memory))
}

// Stat fills a metrics structure with usage stats for the controller
func (c *linuxMemHandler) Stat(ctr *CgroupControl, m *cgroups.Stats) error {
	var err error
	memUsage := cgroups.MemoryStats{}

	var memoryRoot string
	var limitFilename string

	if ctr.cgroup2 {
		memoryRoot = filepath.Join(cgroupRoot, ctr.config.Path)
		limitFilename = "memory.max"
		if memUsage.Usage.Usage, err = readFileByKeyAsUint64(filepath.Join(memoryRoot, "memory.stat"), "anon"); err != nil {
			return err
		}
	} else {
		memoryRoot = ctr.getCgroupv1Path(Memory)
		limitFilename = "memory.limit_in_bytes"
		if memUsage.Usage.Usage, err = readFileAsUint64(filepath.Join(memoryRoot, "memory.usage_in_bytes")); err != nil {
			return err
		}
	}

	memUsage.Usage.Limit, err = readFileAsUint64(filepath.Join(memoryRoot, limitFilename))
	if err != nil {
		return err
	}

	m.MemoryStats = memUsage
	return nil
}
