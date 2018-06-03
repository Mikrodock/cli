package cluster

import (
	"mikrodock-cli/drivers"
	"mikrodock-cli/logger"
	"mikrodock-cli/provision"
	"mikrodock-cli/utils"
	"mikrodock-cli/utils/certs"
	"mikrodock-cli/utils/mssh"
	"os"
	"path"
)

func generateMinimalFiles(c *Cluster) {
	logger.Info("ClusterInit", "Creating directories")
	if err := c.CreateDirectoryStructure(); err != nil {
		logger.Fatal("ClusterInit", err.Error())
	}
	logger.Info("ClusterInit", "Directories created")

	logger.Info("ClusterInit", "Generating SSH keys")
	privateKeyPath := path.Join(c.SSHPath(), "private_key")
	if err := mssh.CreatePrivateKey(privateKeyPath); err != nil {
		logger.Fatal("ClusterInit", err.Error())
	}
}

func getProvider(driver drivers.Driver) provision.Provider {

	osRelease, stderr, err := driver.SSHCommand("cat /etc/os-release")
	if err != nil {
		logger.Fatal("ClusterInit.Konsultant.SSH", err.Error())
	}
	if stderr != "" {
		logger.Warn("ClusterInit.Konsultant.SSH", stderr)
	}

	osType := utils.DetectOSType(osRelease)

	logger.Info("ClusterIn it.Konsultant.OSDetect", "OS is "+string(osType))

	provider, _ := provision.GetMatchingProvider(osType)
	if provider == nil {
		logger.Fatal("ClusterInit.Konsultant.Provider", "Cannot find a matching provider")
	}

	provider.SetDriver(driver)

	return provider
}

func makeConsulCerts(c *Cluster, driver drivers.Driver) {
	certGen := certs.NewX509CertGenerator()

	err := certGen.GenerateCACert(path.Join(c.ConsulConfPath(), "ca.cert"), path.Join(c.ConsulConfPath(), "ca.key"), "Mikrodock-Consul-CA", 2048)

	if err != nil {
		logger.Fatal("ClusterInit.Security", "Cannot generate CA Certs : "+err.Error())
	}

	opts := &certs.CertOpts{
		CAFile:       path.Join(c.ConsulConfPath(), "ca.cert"),
		CAKeyFile:    path.Join(c.ConsulConfPath(), "ca.key"),
		CertFile:     path.Join(c.ConsulConfPath(), "cert.pem"),
		KeyFile:      path.Join(c.ConsulConfPath(), "key.pem"),
		KeyBits:      2048,
		MainHost:     "consul.mikrodock.local",
		AliasIPs:     []string{driver.GetBaseDriver().IPAddress, "127.0.0.1"},
		AliasHosts:   []string{},
		MasterMode:   true,
		Organization: "Mikrodock-Consul",
	}

	err = certGen.GenerateCert(opts)
	if err != nil {
		logger.Fatal("ClusterInit.Security", "Cannot generate Certs : "+err.Error())
	}

	_, _, err = driver.SSHCommand("mkdir -p /opt/consul-ssl/")
	if err != nil {
		logger.Fatal("ClusterInit.Security", "Cannot generate remote directory : "+err.Error())
	}

	err = driver.CopyFile(path.Join(c.ConsulConfPath(), "ca.cert"), "/opt/consul-ssl//ca.cert")
	if err != nil {
		logger.Fatal("ClusterInit.Security", "Cannot copy certs to host : "+err.Error())
	}

	err = driver.CopyFile(path.Join(c.ConsulConfPath(), "cert.pem"), "/opt/consul-ssl/cert.pem")
	if err != nil {
		logger.Fatal("ClusterInit.Security", "Cannot copy certs to host : "+err.Error())
	}

	err = driver.CopyFile(path.Join(c.ConsulConfPath(), "key.pem"), "/opt/consul-ssl/key.pem")
	if err != nil {
		logger.Fatal("ClusterInit.Security", "Cannot copy certs to host : "+err.Error())
	}
}

func makeCA(c *Cluster) {
	certGen := certs.NewX509CertGenerator()

	err := certGen.GenerateCACert(path.Join(c.DockerConfigPath(), "ca.cert"), path.Join(c.DockerConfigPath(), "ca.key"), "Mikrodock-CA", 2048)

	if err != nil {
		logger.Fatal("ClusterInit.Security", "Cannot generate CA Certs : "+err.Error())
	}
}

func makeCerts(c *Cluster, driver drivers.Driver) {

	pDir := c.PartiklePath(driver.GetBaseDriver().MachineName)
	if err := os.MkdirAll(pDir, 0775); err != nil {
		logger.Fatal("ClusterInit.Security", "Cannot generate Certs Dir : "+err.Error())
	}

	certGen := certs.NewX509CertGenerator()

	opts := &certs.CertOpts{
		CAFile:       path.Join(c.DockerConfigPath(), "ca.cert"),
		CAKeyFile:    path.Join(c.DockerConfigPath(), "ca.key"),
		CertFile:     path.Join(c.PartiklePath(driver.GetBaseDriver().MachineName), "cert.pem"),
		KeyFile:      path.Join(c.PartiklePath(driver.GetBaseDriver().MachineName), "key.pem"),
		KeyBits:      2048,
		MainHost:     "konsultant.mikrodock.local",
		AliasIPs:     []string{driver.GetBaseDriver().IPAddress, "127.0.0.1"},
		AliasHosts:   []string{},
		MasterMode:   true,
		Organization: "Mikrodock",
	}

	err := certGen.GenerateCert(opts)
	if err != nil {
		logger.Fatal("ClusterInit.Security", "Cannot generate Certs : "+err.Error())
	}

	_, _, err = driver.SSHCommand("mkdir -p /etc/docker")
	if err != nil {
		logger.Fatal("ClusterInit.Security", "Cannot generate remote directory : "+err.Error())
	}

	err = driver.CopyFile(path.Join(c.DockerConfigPath(), "ca.cert"), "/etc/docker/ca.cert")
	if err != nil {
		logger.Fatal("ClusterInit.Security", "Cannot copy certs to host : "+err.Error())
	}

	err = driver.CopyFile(path.Join(c.PartiklePath(driver.GetBaseDriver().MachineName), "cert.pem"), "/etc/docker/cert.pem")
	if err != nil {
		logger.Fatal("ClusterInit.Security", "Cannot copy certs to host : "+err.Error())
	}

	err = driver.CopyFile(path.Join(c.PartiklePath(driver.GetBaseDriver().MachineName), "key.pem"), "/etc/docker/key.pem")
	if err != nil {
		logger.Fatal("ClusterInit.Security", "Cannot copy certs to host : "+err.Error())
	}
}

func copyConsulCerts(c *Cluster, driver drivers.Driver) {
	_, _, err := driver.SSHCommand("mkdir -p /etc/docker/")
	if err != nil {
		logger.Fatal("ClusterInit.Security", "Cannot generate remote directory : "+err.Error())
	}

	err = driver.CopyFile(path.Join(c.ConsulConfPath(), "ca.cert"), "/etc/docker/kv-ca.pem")
	if err != nil {
		logger.Fatal("ClusterInit.Security", "Cannot copy certs to host : "+err.Error())
	}

	err = driver.CopyFile(path.Join(c.ConsulConfPath(), "cert.pem"), "/etc/docker/kv-cert.pem")
	if err != nil {
		logger.Fatal("ClusterInit.Security", "Cannot copy certs to host : "+err.Error())
	}

	err = driver.CopyFile(path.Join(c.ConsulConfPath(), "key.pem"), "/etc/docker/kv-key.pem")
	if err != nil {
		logger.Fatal("ClusterInit.Security", "Cannot copy certs to host : "+err.Error())
	}
}

func configureDocker(provider provision.Provider, extraConf string) {

	hasDocker, err := provider.DetectDocker()

	if err != nil {
		logger.Fatal("ClusterInit.Konsultant.Docker", err.Error())
	}

	if hasDocker {
		logger.Info("ClusterInit.Konsultant.Docker", "Docker has been sucessfully detected")
	} else {
		logger.Info("ClusterInit.Konsultant.Docker", "No Docker installation detected")

		// TODO : Docker installation
	}

	err = provider.StopDocker()
	if err != nil {
		logger.Fatal("ClusterInit.Konsultant.Docker", "Cannot stop docker : "+err.Error())
	}

	err = provider.ConfigureDocker(extraConf)
	if err != nil {
		logger.Fatal("ClusterInit.Konsultant.Docker", "Cannot configure docker : "+err.Error())
	}

	err = provider.StartDocker()
	if err != nil {
		logger.Fatal("ClusterInit.Konsultant.Docker", "Cannot start docker : "+err.Error())
	}
}
