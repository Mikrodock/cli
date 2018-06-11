package cluster

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"mikrodock-cli/drivers"
	"mikrodock-cli/logger"
	"mikrodock-cli/provision"
	"mikrodock-cli/utils/certs"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	consulAPI "github.com/hashicorp/consul/api"
)

type DockerClusterOptions struct {
	ClusterStoreAddress string
	AdvertiseAddress    string
	CAPath              string
	CertPath            string
	KeyPath             string
}

func (d *DockerClusterOptions) String() string {
	var sb bytes.Buffer
	if len(d.ClusterStoreAddress) != 0 {
		sb.WriteString("--cluster-store=consul://" + d.ClusterStoreAddress + "\n")
	}
	if len(d.AdvertiseAddress) != 0 {
		sb.WriteString("--cluster-advertise=" + d.AdvertiseAddress + "\n")
	}
	if len(d.CAPath) != 0 {
		sb.WriteString("--cluster-store-opt kv.cacertfile=" + d.CAPath + "\n")
	}
	if len(d.CertPath) != 0 {
		sb.WriteString("--cluster-store-opt kv.certfile=" + d.CertPath + "\n")
	}
	if len(d.KeyPath) != 0 {
		sb.WriteString("--cluster-store-opt kv.keyfile=" + d.KeyPath + "\n")
	}
	fmt.Println(sb.String())
	return sb.String()
}

type Partikle struct {
	Driver   drivers.Driver
	Provider provision.Provider
	Galaksy  *Cluster
	IsMaster bool
}

func newEmptyPartikle() *Partikle {
	return &Partikle{}
}

func NewPartikle(driver drivers.Driver, provider provision.Provider, c *Cluster) *Partikle {
	return &Partikle{
		Driver:   driver,
		Provider: provider,
		Galaksy:  c,
	}
}

func (p *Partikle) Name() string {
	return p.Driver.GetBaseDriver().MachineName
}

func (p *Partikle) Path() string {
	return path.Join(p.Galaksy.PartiklePath(p.Name()))
}

func (p *Partikle) CertsPath() string {
	return path.Join(p.Path(), "certs")
}

func (p *Partikle) Create() error {
	return p.Driver.Create()
}

func (p *Partikle) GenerateConsulCerts(caDir string) error {
	certGen := certs.NewX509CertGenerator()

	err := certGen.GenerateCACert(path.Join(caDir, "ca.cert"), path.Join(caDir, "ca.key"), "Mikrodock-Consul-CA", 2048)

	if err != nil {
		return err
	}

	opts := &certs.CertOpts{
		CAFile:       path.Join(p.Galaksy.ConsulConfPath(), "ca.cert"),
		CAKeyFile:    path.Join(p.Galaksy.ConsulConfPath(), "ca.key"),
		CertFile:     path.Join(p.Galaksy.ConsulConfPath(), "cert.pem"),
		KeyFile:      path.Join(p.Galaksy.ConsulConfPath(), "key.pem"),
		KeyBits:      2048,
		MainHost:     "consul.mikrodock.local",
		AliasIPs:     []string{p.Driver.GetBaseDriver().IPAddress, "127.0.0.1"},
		AliasHosts:   []string{},
		MasterMode:   true,
		Organization: "Mikrodock-Consul",
	}

	return certGen.GenerateCert(opts)
}

func (p *Partikle) GenerateDockerCerts(caDir string) error {
	if _, err := os.Stat(p.CertsPath()); os.IsNotExist(err) {
		if err := os.MkdirAll(p.CertsPath(), 0775); err != nil {
			logger.Fatal("ClusterInit.Security", "Cannot generate Certs Dir : "+err.Error())
		}
	}

	opts := &certs.CertOpts{
		CAFile:       path.Join(caDir, "ca.cert"),
		CAKeyFile:    path.Join(caDir, "ca.key"),
		CertFile:     path.Join(p.CertsPath(), "cert.pem"),
		KeyFile:      path.Join(p.CertsPath(), "key.pem"),
		KeyBits:      2048,
		MainHost:     p.Name() + ".mikrodock.local",
		AliasIPs:     []string{p.Driver.GetBaseDriver().IPAddress, "127.0.0.1"},
		AliasHosts:   []string{},
		MasterMode:   true,
		Organization: "Mikrodock",
	}

	certGen := certs.NewX509CertGenerator()

	err := certGen.GenerateCert(opts)

	return err
}

func (p *Partikle) UploadConsulCerts(remotePath string) error {

	err := p.Mkdir(remotePath)
	if err != nil {
		return err
	}

	err = p.UploadFile(path.Join(p.Galaksy.ConsulConfPath(), "ca.cert"), path.Join(remotePath, "kv-ca.cert"))
	if err != nil {
		return err
	}

	err = p.UploadFile(path.Join(p.Galaksy.ConsulConfPath(), "cert.pem"), path.Join(remotePath, "kv-cert.pem"))
	if err != nil {
		return err
	}

	err = p.UploadFile(path.Join(p.Galaksy.ConsulConfPath(), "key.pem"), path.Join(remotePath, "kv-key.pem"))

	return err
}

func (p *Partikle) UploadDockerCerts() error {
	err := p.Mkdir("/etc/docker")
	if err != nil {
		return err
	}

	caDir := p.Galaksy.DockerConfigPath()

	if err = p.UploadFile(path.Join(caDir, "ca.cert"), "/etc/docker/ca.cert"); err != nil {
		return err
	}

	if err = p.UploadFile(path.Join(p.CertsPath(), "cert.pem"), "/etc/docker/cert.pem"); err != nil {
		return err
	}

	return p.UploadFile(path.Join(p.CertsPath(), "key.pem"), "/etc/docker/key.pem")

}

func (p *Partikle) RunConsulContainer() error {

	vols := make(map[string]struct{})
	vols["/consul/data"] = struct{}{}
	vols["/consul/ssl"] = struct{}{}

	return p.RunContainer("izanagi1995/consul-ssl", "mikro-consul", vols, &container.Config{
		Hostname: "mikro-consul",
		Image:    "izanagi1995/consul-ssl",
		Env:      []string{"CONSUL_LOCAL_CONFIG={\"skip_leave_on_interrupt\": true, \"addresses\": {\"https\": \"" + p.Driver.GetBaseDriver().IPAddress + "\"}, \"ports\" : {\"https\" : 8081, \"http\": -1}, \"ca_file\": \"/consul/ssl/kv-ca.cert\", \"cert_file\": \"/consul/ssl/kv-cert.pem\", \"key_file\": \"/consul/ssl/kv-key.pem\", \"verify_outgoing\": true, \"verify_incoming\": true}"},
		Cmd:      []string{"consul", "agent", "-server", "-data-dir=/consul/data", "-bind=" + p.Driver.GetBaseDriver().IPAddress, "-client=" + p.Driver.GetBaseDriver().IPAddress, "-config-dir=/consul/config", "-bootstrap"},
		Volumes:  vols,
	}, &container.HostConfig{
		Binds:       []string{"/opt/consul:/consul/data", "/opt/consul-ssl/:/consul/ssl"},
		NetworkMode: "host",
	}, &network.NetworkingConfig{})
}

func (p *Partikle) RunContainer(imageName string, containerName string, volumes map[string]struct{}, containerConfig *container.Config, hostConfig *container.HostConfig, netConfig *network.NetworkingConfig) error {
	dockerClient, err := p.NewDockerClient()
	if err != nil {
		return err
	}

	reader, err := dockerClient.ImagePull(context.Background(), imageName, types.ImagePullOptions{})
	if err != nil {
		return err
	}
	ioutil.ReadAll(reader)

	body, err := dockerClient.ContainerCreate(context.Background(), containerConfig, hostConfig, netConfig, containerName)

	if err != nil {
		logger.Fatal("ClusterInit.Konsultant.Docker", "Cannot create consul container : "+err.Error())
	}

	return dockerClient.ContainerStart(context.Background(), body.ID, types.ContainerStartOptions{})
}

func (p *Partikle) ConnectToConsul() (*consulAPI.Client, error) {
	consulConfig := consulAPI.DefaultConfig()
	consulConfig.Address = p.Driver.GetBaseDriver().IPAddress + ":8081"
	consulConfig.Scheme = "https"

	consulConfig.TLSConfig = consulAPI.TLSConfig{
		Address:            p.Driver.GetBaseDriver().IPAddress + ":8081",
		CAFile:             path.Join(p.Galaksy.ConsulConfPath(), "ca.cert"),
		CertFile:           path.Join(p.Galaksy.ConsulConfPath(), "cert.pem"),
		KeyFile:            path.Join(p.Galaksy.ConsulConfPath(), "key.pem"),
		InsecureSkipVerify: true,
	}

	client, err := consulAPI.NewClient(consulConfig)
	if err != nil {
		return nil, err
	}

	// Trick to check if the connection is ready
	_, _, err = client.KV().List("", nil)
	retries := 0

	for err != nil && retries < 10 {
		time.Sleep(5 * time.Second)
		_, _, err = client.KV().List("", nil)
		retries++
	}

	if retries == 10 {
		return nil, errors.New("Cannot connect to Consul (Timeout after 50 seconds)")
	}

	return client, nil
}

func (p *Partikle) UploadFile(source string, destination string) error {
	return p.Driver.CopyFile(source, destination)
}

func (p *Partikle) DetectDocker() (bool, error) {
	return p.Provider.DetectDocker()
}

func (p *Partikle) ConfigureEnv(envs map[string]string) {
	for key, value := range envs {
		p.Driver.SSHCommand(fmt.Sprintf("echo 'export %s=%s' >> ~/.env", key, value))
	}
	p.Driver.SSHCommand("echo 'source ~/.env'")
}

func (p *Partikle) InstallDocker() error {
	return p.Provider.InstallDocker()
}

func (p *Partikle) StopDocker() error {
	return p.Provider.StopDocker()
}

func (p *Partikle) StartDocker() error {
	return p.Provider.StartDocker()
}

func (p *Partikle) ConfigureDocker(d *DockerClusterOptions) error {

	if err := p.StopDocker(); err != nil {
		return err
	}

	if d != nil {
		if err := p.Provider.ConfigureDocker(d.String()); err != nil {
			return err
		}
	} else {
		if err := p.Provider.ConfigureDocker(""); err != nil {
			return err
		}
	}

	return p.StartDocker()
}

func (p *Partikle) NewDockerClient() (*client.Client, error) {
	return p.Provider.GetBaseProvider().ConnectDocker(p.CertsPath(), p.Galaksy.DockerConfigPath())
}

func (p *Partikle) Mkdir(path string) error {
	return p.Provider.CreateDirectory(path)
}

func (p *Partikle) IP() string {
	return p.Driver.GetBaseDriver().IPAddress
}

func (p *Partikle) WaitDocker() error {
	client, err := p.NewDockerClient()
	if err != nil {
		return err
	}

	_, err = client.ContainerList(context.Background(), types.ContainerListOptions{})

	for i := 0; i < 6; i++ {
		if err == nil {
			return nil
		}
		time.Sleep(10 * time.Second)
		_, err = client.ContainerList(context.Background(), types.ContainerListOptions{})
	}

	return err
}

func (p *Partikle) Save() error {
	savePath := path.Join(p.Path(), "data.mk")
	file, err := os.Create(savePath)
	defer file.Close()
	if err == nil {
		var buffer bytes.Buffer
		buffer.WriteString(p.Driver.GetBaseDriver().IPAddress + "\n")
		buffer.WriteString(p.Driver.GetBaseDriver().MachineName + "\n")
		buffer.WriteString(p.Driver.GetBaseDriver().SSHKeyPath + "\n")
		buffer.WriteString(p.Driver.GetBaseDriver().SSHPort + "\n")
		buffer.WriteString(p.Driver.GetBaseDriver().SSHUser + "\n")
		buffer.WriteString(strconv.Itoa(p.Driver.GetBaseDriver().MachineID) + "\n")
		buffer.WriteString(strconv.FormatBool(p.IsMaster) + "\n")

		file.Write(buffer.Bytes())
	}

	return err
}

func LoadPartikle(Gal *Cluster, Name string) (*Partikle, error) {
	var part *Partikle
	pPath := path.Join(Gal.PartiklePath(Name), "data.mk")
	file, err := os.Open(pPath)
	defer file.Close()

	driverConfig := make(map[string]interface{})
	driverConfig["ssh-key-path"] = path.Join(Gal.SSHPath(), "private_key")
	driverConfig["name"] = path.Join(Gal.SSHPath(), Name)

	driver, _ := Gal.DriverFactory(driverConfig)
	base := drivers.BaseDriver{}

	if err == nil {
		scanner := bufio.NewScanner(file)

		scanner.Scan()
		base.IPAddress = scanner.Text()
		scanner.Scan()
		base.MachineName = scanner.Text()
		scanner.Scan()
		base.SSHKeyPath = scanner.Text()
		scanner.Scan()
		base.SSHPort = scanner.Text()
		scanner.Scan()
		base.SSHUser = scanner.Text()
		scanner.Scan()
		base.MachineID, _ = strconv.Atoi(scanner.Text())

		driver.SetBaseDriver(base)

		fmt.Printf("%#v\n", driver)

		provider := getProvider(driver)

		part = NewPartikle(driver, provider, Gal)
		scanner.Scan()
		part.IsMaster, _ = strconv.ParseBool(scanner.Text())

	}

	return part, err
}
