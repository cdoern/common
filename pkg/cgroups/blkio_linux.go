//go:build linux
// +build linux

package cgroups

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/opencontainers/runc/libcontainer/cgroups"
	"github.com/opencontainers/runc/libcontainer/configs"
	"github.com/pkg/errors"
)

type linuxBlkioHandler struct {
}

func getBlkioHandler() *linuxBlkioHandler {
	return &linuxBlkioHandler{}
}

// Apply set the specified constraints
func (c *linuxBlkioHandler) Apply(ctr *CgroupControl, res *configs.Resources) error {
	if ctr.cgroup2 {
		path := filepath.Join(cgroupRoot, ctr.config.Path)
		var bfq *os.File
		// This has to do with checking for the bfq handler
		if res.BlkioWeight != 0 || len(res.BlkioWeightDevice) > 0 {
			var err error
			bfq, err = OpenFile(path, "io.bfq.weight", os.O_RDWR)
			if err == nil {
				defer bfq.Close()
			} else if !os.IsNotExist(err) {
				return err
			}
		}

		if res.BlkioWeight != 0 {
			if bfq != nil {
				if _, err := bfq.WriteString(strconv.FormatUint(uint64(res.BlkioWeight), 10)); err != nil {
					return err
				}
			} else {
				var v uint64
				if res.BlkioWeight != 0 {
					v = 1 + (uint64(res.BlkioWeight)-10)*9999/990
				} else {
					v = 0
				}
				if err := WriteFile(path, "io.weight", strconv.FormatUint(v, 10)); err != nil {
					return err
				}
			}
		}
		if bfqDeviceWeightSupported(bfq) {
			for _, wd := range res.BlkioWeightDevice {
				if _, err := bfq.WriteString(wd.WeightString() + "\n"); err != nil {
					return fmt.Errorf("setting device weight %q: %w", wd.WeightString(), err)
				}
			}
		}
		for _, td := range res.BlkioThrottleReadBpsDevice {
			if err := WriteFile(path, "io.max", td.StringName("rbps")); err != nil {
				return err
			}
		}
		for _, td := range res.BlkioThrottleWriteBpsDevice {
			if err := WriteFile(path, "io.max", td.StringName("wbps")); err != nil {
				return err
			}
		}
		for _, td := range res.BlkioThrottleReadIOPSDevice {
			if err := WriteFile(path, "io.max", td.StringName("riops")); err != nil {
				return err
			}
		}
		for _, td := range res.BlkioThrottleWriteIOPSDevice {
			if err := WriteFile(path, "io.max", td.StringName("wiops")); err != nil {
				return err
			}
		}
		return nil
	}
	// maintaining the fs2 and fs1 functions here for future development
	path := filepath.Join(cgroupRoot, Blkio, ctr.config.Path)
	weightFile, weightDeviceFile := GetBlkioFiles(path)
	if res.BlkioWeight != 0 {
		if err := WriteFile(path, weightFile, strconv.FormatUint(uint64(res.BlkioWeight), 10)); err != nil {
			return err
		}
	}
	if res.BlkioLeafWeight != 0 {
		if err := WriteFile(path, "blkio.leaf_weight", strconv.FormatUint(uint64(res.BlkioLeafWeight), 10)); err != nil {
			return err
		}
	}
	for _, wd := range res.BlkioWeightDevice {
		if wd.Weight != 0 {
			if err := WriteFile(path, weightDeviceFile, fmt.Sprintf("%d:%d %d", wd.Major, wd.Minor, wd.Weight)); err != nil {
				return err
			}
		}
		if wd.LeafWeight != 0 {
			if err := WriteFile(path, "blkio.leaf_weight_device", fmt.Sprintf("%d:%d %d", wd.Major, wd.Minor, wd.LeafWeight)); err != nil {
				return err
			}
		}
	}
	err := SetBlkioThrottle(res, path)
	if err != nil {
		return err
	}
	return nil
}

// Create the cgroup
func (c *linuxBlkioHandler) Create(ctr *CgroupControl) (bool, error) {
	return ctr.createCgroupDirectory(Blkio)
}

// Destroy the cgroup
func (c *linuxBlkioHandler) Destroy(ctr *CgroupControl) error {
	return rmDirRecursively(ctr.getCgroupv1Path(Blkio))
}

// Stat fills a metrics structure with usage stats for the controller
func (c *linuxBlkioHandler) Stat(ctr *CgroupControl, m *cgroups.Stats) error {
	var ioServiceBytesRecursive []cgroups.BlkioStatEntry

	if ctr.cgroup2 {
		// more details on the io.stat file format:X https://facebookmicrosites.github.io/cgroup2/docs/io-controller.html
		values, err := readCgroup2MapFile(ctr, "io.stat")
		if err != nil {
			return err
		}
		for k, v := range values {
			d := strings.Split(k, ":")
			if len(d) != 2 {
				continue
			}
			minor, err := strconv.ParseUint(d[0], 10, 0)
			if err != nil {
				return err
			}
			major, err := strconv.ParseUint(d[1], 10, 0)
			if err != nil {
				return err
			}

			for _, item := range v {
				d := strings.Split(item, "=")
				if len(d) != 2 {
					continue
				}
				op := d[0]

				// Accommodate the cgroup v1 naming
				switch op {
				case "rbytes":
					op = "read"
				case "wbytes":
					op = "write"
				}

				value, err := strconv.ParseUint(d[1], 10, 0)
				if err != nil {
					return err
				}

				entry := cgroups.BlkioStatEntry{
					Op:    op,
					Major: major,
					Minor: minor,
					Value: value,
				}
				ioServiceBytesRecursive = append(ioServiceBytesRecursive, entry)
			}
		}
	} else {
		BlkioRoot := ctr.getCgroupv1Path(Blkio)

		p := filepath.Join(BlkioRoot, "blkio.throttle.io_service_bytes_recursive")
		f, err := os.Open(p)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return errors.Wrapf(err, "open %s", p)
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			parts := strings.Fields(line)
			if len(parts) < 3 {
				continue
			}
			d := strings.Split(parts[0], ":")
			if len(d) != 2 {
				continue
			}
			minor, err := strconv.ParseUint(d[0], 10, 0)
			if err != nil {
				return err
			}
			major, err := strconv.ParseUint(d[1], 10, 0)
			if err != nil {
				return err
			}

			op := parts[1]

			value, err := strconv.ParseUint(parts[2], 10, 0)
			if err != nil {
				return err
			}
			entry := cgroups.BlkioStatEntry{
				Op:    op,
				Major: major,
				Minor: minor,
				Value: value,
			}
			ioServiceBytesRecursive = append(ioServiceBytesRecursive, entry)
		}
		if err := scanner.Err(); err != nil {
			return errors.Wrapf(err, "parse %s", p)
		}
	}
	m.BlkioStats.IoServiceBytesRecursive = ioServiceBytesRecursive
	return nil
}
