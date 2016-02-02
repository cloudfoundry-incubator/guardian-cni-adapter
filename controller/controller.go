package controller

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/appc/cni/libcni"
)

type Controller struct {
	PluginDir string
	ConfigDir string

	cniConfig      *libcni.CNIConfig
	networkConfigs []*libcni.NetworkConfig
}

func (c *Controller) ensureInitialized() error {
	if c.cniConfig == nil {
		c.cniConfig = &libcni.CNIConfig{Path: []string{c.PluginDir}}
	}

	if c.networkConfigs == nil {
		c.networkConfigs = []*libcni.NetworkConfig{}

		err := filepath.Walk(c.ConfigDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, ".conf") {
				return nil
			}

			conf, err := libcni.ConfFromFile(path)
			if err != nil {
				return fmt.Errorf("unable to load config from %s: %s", path, err)
			}
			c.networkConfigs = append(c.networkConfigs, conf)
			log.Printf("loaded config %+v\n%s\n", conf.Network, string(conf.Bytes))
			return nil
		})
		if err != nil {
			return fmt.Errorf("error loading config: %s", err)
		}
	}

	return nil
}

func (c *Controller) Up(namespacePath, handle, spec string) error {
	err := c.ensureInitialized()
	if err != nil {
		return fmt.Errorf("failed to initialize controller: %s", err)
	}

	for i, networkConfig := range c.networkConfigs {
		runtimeConfig := &libcni.RuntimeConf{
			ContainerID: handle,
			NetNS:       namespacePath,
			IfName:      fmt.Sprintf("eth%d", i),
		}
		_, err = c.cniConfig.AddNetwork(networkConfig, runtimeConfig)
		if err != nil {
			return fmt.Errorf("add network failed: %s", err)
		}
	}

	return nil
}

func (c *Controller) Down(namespacePath, handle string) error {
	err := c.ensureInitialized()
	if err != nil {
		return fmt.Errorf("failed to initialize controller: %s", err)
	}

	for i, networkConfig := range c.networkConfigs {
		runtimeConfig := &libcni.RuntimeConf{
			ContainerID: handle,
			NetNS:       namespacePath,
			IfName:      fmt.Sprintf("eth%d", i),
		}
		err = c.cniConfig.DelNetwork(networkConfig, runtimeConfig)
		if err != nil {
			return fmt.Errorf("add network failed: %s", err)
		}
	}

	return nil
}
