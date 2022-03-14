//go:build linux
// +build linux

package cgroups

import (
	"testing"

	"github.com/containers/storage/pkg/unshare"
	"github.com/opencontainers/runc/libcontainer/configs"
)

func TestCreated(t *testing.T) {
	// tests only works in rootless mode
	if unshare.IsRootless() {
		return
	}

	var resources configs.Resources
	cgr, err := New("machine.slice", &resources)
	if err != nil {
		t.Fatal(err)
	}
	if err := cgr.Delete(); err != nil {
		t.Fatal(err)
	}

	cgr, err = NewSystemd("machine.slice")
	if err != nil {
		t.Fatal(err)
	}
	if err := cgr.Delete(); err != nil {
		t.Fatal(err)
	}
}

func TestResources(t *testing.T) {
	// tests only works in rootless mode
	if unshare.IsRootless() {
		return
	}

	var resources configs.Resources
	resources.CpuPeriod = 100000
	resources.CpuQuota = 100000

	cgr, err := New("machine.slice", &resources)
	if err != nil {
		t.Fatal(err)
	}

	// TestMode is used in the runc packages for unit tests, works without this as well here.
	TestMode = true
	err = cgr.Update(&resources)
	if err != nil {
		t.Fatal(err)
	}
	if cgr.config.CpuPeriod != 100000 || cgr.config.CpuQuota != 100000 {
		t.Fatal("Got the wrong value, set cpu.cfs_period_us failed.")
	}
}
