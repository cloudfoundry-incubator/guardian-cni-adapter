package controller

import (
	"fmt"
	"path/filepath"
)

//go:generate counterfeiter -o ../fakes/cniController.go --fake-name CNIController . cniController
type cniController interface {
	Up(namespacePath, handle, spec string) error
	Down(namespacePath, handle string) error
}

//go:generate counterfeiter -o ../fakes/mounter.go --fake-name Mounter . mounter
type mounter interface {
	IdempotentlyMount(source, target string) error
	RemoveMount(target string) error
}

type Manager struct {
	CNIController cniController
	Mounter       mounter
	BindMountRoot string
}

func (m *Manager) Up(pid int, containerHandle, networkSpec string) error {
	procNsPath := fmt.Sprintf("/proc/%d/ns/net", pid)
	bindMountPath := filepath.Join(m.BindMountRoot, containerHandle)

	err := m.Mounter.IdempotentlyMount(procNsPath, bindMountPath)
	if err != nil {
		return fmt.Errorf("failed mounting %s to %s: %s", procNsPath, bindMountPath, err)
	}

	err = m.CNIController.Up(bindMountPath, containerHandle, networkSpec)
	if err != nil {
		return fmt.Errorf("cni up failed: %s", err)
	}

	return nil
}

func (m *Manager) Down(containerHandle string) error {
	bindMountPath := filepath.Join(m.BindMountRoot, containerHandle)

	err := m.CNIController.Down(bindMountPath, containerHandle)
	if err != nil {
		return fmt.Errorf("cni down failed: %s", err)
	}

	err = m.Mounter.RemoveMount(bindMountPath)
	if err != nil {
		return fmt.Errorf("failed removing mount %s: %s", bindMountPath, err)
	}

	return nil
}
