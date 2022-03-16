//go:build linux
// +build linux

package cgroups

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/opencontainers/runc/libcontainer/cgroups"
	"github.com/opencontainers/runc/libcontainer/cgroups/fs"
	"github.com/opencontainers/runc/libcontainer/configs"
	"github.com/pkg/errors"
)

type linuxCpuHandler struct {
	Cpu fs.CpuGroup
}

func getCPUHandler() *linuxCpuHandler {
	return &linuxCpuHandler{}
}

// Apply set the specified constraints
func (c *linuxCpuHandler) Apply(ctr *CgroupControl, res *configs.Resources) error {
	if ctr.cgroup2 {
		path := filepath.Join(cgroupRoot, ctr.config.Path)
		if res.CpuWeight != 0 {
			if err := WriteFile(path, "cpu.weight", strconv.FormatUint(res.CpuWeight, 10)); err != nil {
				return err
			}
		}
		if res.CpuQuota != 0 || res.CpuPeriod != 0 {
			str := "max"
			if res.CpuQuota > 0 {
				str = strconv.FormatInt(res.CpuQuota, 10)
			}
			period := res.CpuPeriod
			if period == 0 {
				period = 100000 // sane default value from the kernel
			}
			str += " " + strconv.FormatUint(period, 10)
			if err := WriteFile(path, "cpu.max", str); err != nil {
				return err
			}
		}
		return nil
	}
	// maintaining the fs2 and fs1 functions here for future development
	path := filepath.Join(cgroupRoot, CPU, ctr.config.Path)
	/*
		if res.CpuShares != 0 {
			shares := res.CpuShares
			if err := WriteFile(path, "cpu.shares", strconv.FormatUint(shares, 10)); err != nil {
				return err
			}
			// read it back
			sharesRead, err := fscommon.GetCgroupParamUint(path, "cpu.shares")
			if err != nil {
				return err
			}
			// ... and check
			if shares > sharesRead {
				return fmt.Errorf("the maximum allowed cpu-shares is %d", sharesRead)
			} else if shares < sharesRead {
				return fmt.Errorf("the minimum allowed cpu-shares is %d", sharesRead)
			}
		}

		var period string
		if res.CpuPeriod != 0 {
			period = strconv.FormatUint(res.CpuPeriod, 10)
			if err := WriteFile(path, "cpu.cfs_period_us", period); err != nil {
				if !errors.Is(err, unix.EINVAL) || res.CpuQuota == 0 {
					return err
				}
			} else {
				period = ""
			}
		}
		if res.CpuQuota != 0 {
			if err := WriteFile(path, "cpu.cfs_quota_us", strconv.FormatInt(res.CpuQuota, 10)); err != nil {
				return err
			}
			if period != "" {
				if err := WriteFile(path, "cpu.cfs_period_us", period); err != nil {
					return err
				}
			}
		}

		// rt setting
		if res.CpuRtPeriod != 0 {
			if err := WriteFile(path, "cpu.rt_period_us", strconv.FormatUint(res.CpuRtPeriod, 10)); err != nil {
				return err
			}
		}
		if res.CpuRtRuntime != 0 {
			if err := WriteFile(path, "cpu.rt_runtime_us", strconv.FormatInt(res.CpuRtRuntime, 10)); err != nil {
				return err
			}
		}
		return nil*/
	return c.Cpu.Set(path, res)
}

// Create the cgroup
func (c *linuxCpuHandler) Create(ctr *CgroupControl) (bool, error) {
	return ctr.createCgroupDirectory(CPU)
}

// Destroy the cgroup
func (c *linuxCpuHandler) Destroy(ctr *CgroupControl) error {
	return rmDirRecursively(ctr.getCgroupv1Path(CPU))
}

// Stat fills a metrics structure with usage stats for the controller
func (c *linuxCpuHandler) Stat(ctr *CgroupControl, m *cgroups.Stats) error {
	var err error
	cpu := cgroups.CpuStats{}
	if ctr.cgroup2 {
		values, err := readCgroup2MapFile(ctr, "cpu.stat")
		if err != nil {
			return err
		}
		if val, found := values["usage_usec"]; found {
			cpu.CpuUsage.TotalUsage, err = strconv.ParseUint(cleanString(val[0]), 10, 64)
			if err != nil {
				return err
			}
			cpu.CpuUsage.UsageInKernelmode *= 1000
		}
		if val, found := values["system_usec"]; found {
			cpu.CpuUsage.UsageInKernelmode, err = strconv.ParseUint(cleanString(val[0]), 10, 64)
			if err != nil {
				return err
			}
			cpu.CpuUsage.TotalUsage *= 1000
		}
	} else {
		cpu.CpuUsage.TotalUsage, err = readAcct(ctr, "cpuacct.usage")
		if err != nil {
			if !os.IsNotExist(errors.Cause(err)) {
				return err
			}
			cpu.CpuUsage.TotalUsage = 0
		}
		cpu.CpuUsage.UsageInKernelmode, err = readAcct(ctr, "cpuacct.usage_sys")
		if err != nil {
			if !os.IsNotExist(errors.Cause(err)) {
				return err
			}
			cpu.CpuUsage.UsageInKernelmode = 0
		}
		cpu.CpuUsage.PercpuUsage, err = readAcctList(ctr, "cpuacct.usage_percpu")
		if err != nil {
			if !os.IsNotExist(errors.Cause(err)) {
				return err
			}
			cpu.CpuUsage.PercpuUsage = nil
		}
	}
	m.CpuStats = cpu
	return nil
}
