package fs

import (
	"github.com/opencontainers/runc/libcontainer/cgroups"
	"github.com/opencontainers/runc/libcontainer/configs"
)

type DevicesGroup struct{}

func (s *DevicesGroup) Name() string {
	return "devices"
}

func (s *DevicesGroup) Apply(path string, r *configs.Resources, pid int) error {
	if path == "" {
		// Return error here, since devices cgroup
		// is a hard requirement for container's security.
		return errSubsystemDoesNotExist
	}

	return apply(path, pid)
}

func (s *DevicesGroup) Set(_ string, _ *configs.Resources) error {
	// A shallow implementation that does not set any device access rules.
	return nil
}

func (s *DevicesGroup) GetStats(path string, stats *cgroups.Stats) error {
	return nil
}
