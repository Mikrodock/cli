package cluster

import (
	"fmt"
	"os"
	"path"
)

func (c *Cluster) CreateDirectoryStructure() error {
	if _, err := os.Stat(c.DeployDir); err == nil {
		return fmt.Errorf("The Galaksy deployment directory (%s) already exists", c.DeployDir)
	}
	if err := os.MkdirAll(c.DeployDir, 0775); err != nil {
		return fmt.Errorf("Cannot create Galaksy deployment directory : %s", err.Error())
	}
	partiklesDir := path.Join(c.DeployDir, "partikles")
	if err := os.MkdirAll(partiklesDir, 0775); err != nil {
		return fmt.Errorf("Cannot create partikles directory : %s", err.Error())
	}

	sshDir := path.Join(c.DeployDir, "ssh")
	if err := os.MkdirAll(sshDir, 0775); err != nil {
		return fmt.Errorf("Cannot create ssh config directory : %s", err.Error())
	}

	dockerConfigDir := path.Join(c.DeployDir, "docker")
	if err := os.MkdirAll(dockerConfigDir, 0775); err != nil {
		return fmt.Errorf("Cannot create docker config directory : %s", err.Error())
	}

	consulConfigDir := path.Join(c.DeployDir, "consul")
	if err := os.MkdirAll(consulConfigDir, 0775); err != nil {
		return fmt.Errorf("Cannot create consul config directory : %s", err.Error())
	}

	return nil
}

func (c *Cluster) PartiklePath(name string) string {
	return path.Join(c.DeployDir, "partikles", name)
}

func (c *Cluster) SSHPath() string {
	return path.Join(c.DeployDir, "ssh")
}

func (c *Cluster) DockerConfigPath() string {
	return path.Join(c.DeployDir, "docker")
}

func (c *Cluster) ConsulConfPath() string {
	return path.Join(c.DeployDir, "consul")
}
