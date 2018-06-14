package cluster

import (
	"bufio"
	"bytes"
	"crypto/rsa"
	"fmt"
	"io/ioutil"
	"mikrodock-cli/drivers"
	"mikrodock-cli/logger"
	"mikrodock-cli/utils"
	consulhelpers "mikrodock-cli/utils/consul-helpers"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"strconv"
	"time"

	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	homedir "github.com/mitchellh/go-homedir"
	"golang.org/x/crypto/ssh"

	consulAPI "github.com/hashicorp/consul/api"
)

type ClusterDriver struct {
	DriverName string
	Config     map[string]string
}

type Cluster struct {
	Name          string
	DeployDir     string
	Driver        ClusterDriver
	DriverFactory drivers.InitDriver

	Partikles []*Partikle
}

func LoadCluster(clusterName string) (*Cluster, error) {

	var c *Cluster
	var dir string

	uid := os.Geteuid()
	if uid == 0 {
		realUser := os.Getenv("SUDO_USER")
		if realUser != "" {
			u, _ := user.Lookup(realUser)
			dir = u.HomeDir
		} else {
			dir, _ = homedir.Dir()
		}
	} else {
		dir, _ = homedir.Dir()
	}

	depDir := path.Join(dir, ".mikrodock", clusterName)
	savePath := path.Join(depDir, "data.mk")
	file, err := os.Open(savePath)
	defer file.Close()
	if err == nil {
		scanner := bufio.NewScanner(file)

		scanner.Scan()
		driverName := scanner.Text()
		scanner.Scan()
		token := scanner.Text()

		config := make(map[string]string)
		config["access-token"] = token
		c = &Cluster{
			DeployDir: depDir,
			Driver: ClusterDriver{
				Config:     config,
				DriverName: driverName,
			},
			Name: clusterName,
		}

		initDriver, _ := drivers.NewDriver(c.Driver.DriverName, c.Driver.Config)
		c.DriverFactory = initDriver

		partiklesDir, _ := ioutil.ReadDir(path.Join(c.DeployDir, "partikles"))
		c.Partikles = make([]*Partikle, len(partiklesDir), len(partiklesDir))
		for i, pDir := range partiklesDir {
			p, err := LoadPartikle(c, pDir.Name())
			if err == nil {
				c.Partikles[i] = p
			}
		}
	}
	return c, err
}

func (c *Cluster) Init() {

	generateMinimalFiles(c)

	logger.Info("ClusterInit", "Creating Konsultant Machine")
	initDriver, err := drivers.NewDriver(c.Driver.DriverName, c.Driver.Config)

	if err != nil {
		logger.Fatal("ClusterInit", err.Error())
	}

	c.DriverFactory = initDriver

	driverConfig := make(map[string]interface{})
	driverConfig["ssh-key-path"] = path.Join(c.SSHPath(), "private_key")
	driverConfig["name"] = "konsultant"
	// TODO : EXTENDS

	makeCA(c)

	konsultantDriver, err := c.DriverFactory(driverConfig)
	logger.Info("ClusterInit.Konsultant", "PreCreate OK")
	if err != nil {
		logger.Fatal("ClusterInit.Konsultant", err.Error())
	}
	if err = konsultantDriver.Create(); err != nil {
		logger.Fatal("ClusterInit.Konsultant", err.Error())
	}

	logger.Info("ClusterInit.Konsultant", "Konsultant Machine Created")

	konsultantProvider := getProvider(konsultantDriver)

	konsultant := NewPartikle(konsultantDriver, konsultantProvider, c)

	logger.Info("ClusterInit.Konsultant", "Generating Docker Certs")
	if err = konsultant.GenerateDockerCerts(c.DockerConfigPath()); err != nil {
		logger.Fatal("ClusterInit.Konsultant.Docker", "Cannot generate Docker certs : "+err.Error())
	}
	if err = konsultant.UploadDockerCerts(); err != nil {
		logger.Fatal("ClusterInit.Konsultant.Docker", "Cannot upload Docker certs : "+err.Error())
	}
	logger.Info("ClusterInit.Konsultant", "Generating Consul Certs")
	if err = konsultant.GenerateConsulCerts(c.ConsulConfPath()); err != nil {
		logger.Fatal("ClusterInit.Konsultant.Consul", "Cannot generate Consul certs : "+err.Error())
	}
	if err = konsultant.UploadConsulCerts("/opt/consul-ssl"); err != nil {
		logger.Fatal("ClusterInit.Konsultant.Consul", "Cannot upload Consul certs : "+err.Error())
	}
	if err = konsultant.ConfigureDocker(nil); err != nil {
		logger.Fatal("ClusterInit.Konsultant.Docker", "Cannot configure Docker : "+err.Error())
	}

	if err = konsultant.WaitDocker(); err != nil {
		logger.Fatal("ClusterInit.Konsultant.Docker", "Docker Timeout : "+err.Error())
	}

	if err = konsultant.RunConsulContainer(); err != nil {
		logger.Fatal("ClusterInit.Konsultant.Consul", "Cannot start Consul : "+err.Error())
	}
	consulClient, err := konsultant.ConnectToConsul()

	if err != nil {
		logger.Fatal("ClusterInit.Konsultant.Consul", "Cannot connect to Consul : "+err.Error())
	}

	kvPairName := &consulAPI.KVPair{
		Key:   "mikrodock/cluster/name",
		Value: []byte(c.Name),
	}

	if _, err = consulClient.KV().Put(kvPairName, nil); err != nil {
		logger.Fatal("ClusterInit.Konsultant.Consul", "Cannot bootstrap Consul : "+err.Error())
	}

	logger.Info("ClusterInit.Konsultant", "Konsultant Ready... Now bootstraping Konduktor")

	// ==================================
	// KONSULTANT DONE
	// ==================================

	driverConfig["name"] = "konduktor"

	konduktorDriver, err := c.DriverFactory(driverConfig)
	logger.Info("ClusterInit.Konduktor", "PreCreate OK")
	if err != nil {
		logger.Fatal("ClusterInit.Konduktor", err.Error())
	}
	if err = konduktorDriver.Create(); err != nil {
		logger.Fatal("ClusterInit.Konduktor", err.Error())
	}

	logger.Info("ClusterInit.Konduktor", "Konduktor Machine Created")

	konduktorProvider := getProvider(konduktorDriver)

	konduktor := NewPartikle(konduktorDriver, konduktorProvider, c)
	konduktor.IsMaster = true

	logger.Info("ClusterInit.Konduktor", "Generating Docker Certs")
	if err = konduktor.GenerateDockerCerts(c.DockerConfigPath()); err != nil {
		logger.Fatal("ClusterInit.Konduktor.Docker", "Cannot generate Docker certs : "+err.Error())
	}
	if err = konduktor.UploadDockerCerts(); err != nil {
		logger.Fatal("ClusterInit.Konduktor.Docker", "Cannot upload Docker certs : "+err.Error())
	}

	if err = konduktor.UploadConsulCerts("/etc/docker"); err != nil {
		logger.Fatal("ClusterInit.Konduktor.Consul", "Cannot upload Consul certs : "+err.Error())
	}

	konduktor.UploadFile(filepath.Join(c.SSHPath(), "private_key"), "/root/.ssh/id_rsa")

	envVars := make(map[string]string)
	envVars["CONSUL_IP"] = konsultant.IP() + ":8081"
	envVars["DO_TOKEN"] = c.Driver.Config["access-token"]

	konduktor.ConfigureEnv(envVars)
	logs, errOut, err := konduktor.Driver.SSHCommand("wget https://nsurleraux.be/kinetik-server -O /usr/bin/kinetik-server")
	if err != nil {
		logger.Fatal("ClusterInit.Konduktor.Kinetik", err.Error())
	} else {
		logger.Info("ClusterInit.Konduktor.Kinetik", fmt.Sprintf("%s\n%s", logs, errOut))
	}
	logs, errOut, err = konduktor.Driver.SSHCommand("chmod +x /usr/bin/kinetik-server")
	if err != nil {
		logger.Fatal("ClusterInit.Konduktor.Kinetik", err.Error())
	} else {
		logger.Info("ClusterInit.Konduktor.Kinetik", fmt.Sprintf("%s\n%s", logs, errOut))
	}
	logs, errOut, err = konduktor.Driver.SSHCommand("kinetik-server install")
	if err != nil {
		logger.Fatal("ClusterInit.Konduktor.Kinetik", err.Error())
	} else {
		logger.Info("ClusterInit.Konduktor.Kinetik", fmt.Sprintf("%s\n%s", logs, errOut))
	}

	logs, errOut, err = konduktor.Driver.SSHCommand("kinetik-server start")
	if err != nil {
		logger.Fatal("ClusterInit.Konduktor.Kinetik", err.Error())
	} else {
		logger.Info("ClusterInit.Konduktor.Kinetik", fmt.Sprintf("%s\n%s", logs, errOut))
	}

	if err = konduktor.ConfigureDocker(&DockerClusterOptions{
		AdvertiseAddress:    konduktor.IP() + ":2376",
		ClusterStoreAddress: konsultant.IP() + ":8081",
		CAPath:              "/etc/docker/kv-ca.cert",
		CertPath:            "/etc/docker/kv-cert.pem",
		KeyPath:             "/etc/docker/kv-key.pem",
	}); err != nil {
		logger.Fatal("ClusterInit.Konduktor.Docker", "Cannot configure Docker : "+err.Error())
	}

	if err = konduktor.WaitDocker(); err != nil {
		logger.Fatal("ClusterInit.Konsultant.Docker", "Docker Timeout : "+err.Error())
	}

	if err != nil {
		logger.Fatal("ClusterInit.Konsultant.Registrator", "Cannot create Registrator : "+err.Error())
	}

	// ==================================
	// KONDUKTOR DONE
	// ==================================

	driverConfig["name"] = "klerk"

	klerkDriver, err := c.DriverFactory(driverConfig)

	logger.Info("ClusterInit.Klerk", "PreCreate OK")
	if err != nil {
		logger.Fatal("ClusterInit.Klerk", err.Error())
	}
	if err = klerkDriver.Create(); err != nil {
		logger.Fatal("ClusterInit.Klerk", err.Error())
	}

	logger.Info("ClusterInit.Klerk", "Klerk Machine Created")

	klerkProvider := getProvider(klerkDriver)

	klerk := NewPartikle(klerkDriver, klerkProvider, c)

	logger.Info("ClusterInit.Klerk", "Generating Docker Certs")
	if err = klerk.GenerateDockerCerts(c.DockerConfigPath()); err != nil {
		logger.Fatal("ClusterInit.Klerk.Docker", "Cannot generate Docker certs : "+err.Error())
	}
	if err = klerk.UploadDockerCerts(); err != nil {
		logger.Fatal("ClusterInit.Klerk.Docker", "Cannot upload Docker certs : "+err.Error())
	}

	if err = klerk.UploadConsulCerts("/etc/docker"); err != nil {
		logger.Fatal("ClusterInit.Klerk.Consul", "Cannot upload Consul certs : "+err.Error())
	}

	envVars = make(map[string]string)
	envVars["CONSUL_IP"] = konsultant.IP() + ":8081"
	envVars["KINETIK_MASTER"] = konduktor.IP() + ":10513"

	klerk.ConfigureEnv(envVars)
	logs, errOut, err = klerk.Driver.SSHCommand("wget https://nsurleraux.be/kinetik-client -O /usr/bin/kinetik-client")
	if err != nil {
		logger.Fatal("ClusterInit.Klerk.Kinetik.Download", err.Error())
	} else {
		logger.Info("ClusterInit.Konduktor.Kinetik", fmt.Sprintf("%s\n%s", logs, errOut))
	}
	logs, errOut, err = klerk.Driver.SSHCommand("chmod +x /usr/bin/kinetik-client")
	if err != nil {
		logger.Fatal("ClusterInit.Klerk.Kinetik.Chmod", err.Error())
	} else {
		logger.Info("ClusterInit.Konduktor.Kinetik", fmt.Sprintf("%s\n%s", logs, errOut))
	}
	logs, errOut, err = klerk.Driver.SSHCommand("kinetik-client install")
	if err != nil {
		logger.Fatal("ClusterInit.Klerk.Kinetik.Install", err.Error())
	} else {
		logger.Info("ClusterInit.Konduktor.Kinetik", fmt.Sprintf("%s\n%s", logs, errOut))
	}

	logs, errOut, err = klerk.Driver.SSHCommand("kinetik-client start")
	if err != nil {
		logger.Fatal("ClusterInit.Klerk.Kinetik.Start", err.Error())
	} else {
		logger.Info("ClusterInit.Konduktor.Kinetik", fmt.Sprintf("%s\n%s", logs, errOut))
	}

	if err = klerk.ConfigureDocker(&DockerClusterOptions{
		AdvertiseAddress:    klerk.IP() + ":2376",
		ClusterStoreAddress: konsultant.IP() + ":8081",
		CAPath:              "/etc/docker/kv-ca.cert",
		CertPath:            "/etc/docker/kv-cert.pem",
		KeyPath:             "/etc/docker/kv-key.pem",
	}); err != nil {
		logger.Fatal("ClusterInit.Klerk.Docker", "Cannot configure Docker : "+err.Error())
	}

	time.Sleep(30 * time.Second)

	// ==================================
	// KLERK DONE
	// ==================================

	logger.Info("ClusterInit.Machines", "All machines created...")

	konduktorDocker, err := konduktor.NewDockerClient()

	if err != nil {
		logger.Fatal("ClusterInit.Konduktor.Docker", "Cannot connect to Docker : "+err.Error())
	}

	netLabels := make(map[string]string)
	netLabels["be.mikrodock.network"] = "overlay"

	_, err = konduktorDocker.NetworkCreate(context.Background(), "mikroverlay", types.NetworkCreate{
		Driver:     "overlay",
		Attachable: false,
		Labels:     netLabels,
		IPAM: &network.IPAM{
			Driver: "default",
			Config: []network.IPAMConfig{
				network.IPAMConfig{
					Subnet:  "172.142.0.0/16",
					Gateway: "172.142.0.1",
				},
			},
		},
	})

	if err != nil {
		logger.Fatal("ClusterInit.Konduktor.Docker", "Cannot create overlay network : "+err.Error())
	}

	c.Partikles = make([]*Partikle, 3, 3)
	c.Partikles[0] = konsultant
	c.Partikles[1] = konduktor
	c.Partikles[2] = klerk

	c.Save()

	helper := consulhelpers.NewConsulHelper(consulClient)
	tree := helper.NewTree("mikrodock")
	nodes := tree.AddSubCategory("nodes")
	tree.AddSubCategory("services")

	konsultantConsulTree := nodes.AddSubCategory(konsultant.IP())

	konsultantConsulTree.AddChild("name", []byte("konsultant"))
	konsultantConsulTree.AddChild("type", []byte("KONSULTANT"))

	bytes2736 := []byte(strconv.Itoa(2376))

	konsultantConsulTree.AddChild("docker-port", bytes2736)

	konduktorConsulTree := nodes.AddSubCategory(konduktor.IP())
	konduktorConsulTree.AddChild("name", []byte("kondukotr"))
	konduktorConsulTree.AddChild("type", []byte("KONDUKTOR"))
	konduktorConsulTree.AddChild("docker-port", bytes2736)

	klerkConsulTree := nodes.AddSubCategory(klerk.IP())
	klerkConsulTree.AddChild("name", []byte("klerk"))
	klerkConsulTree.AddChild("type", []byte("KLERK"))
	klerkConsulTree.AddChild("docker-port", bytes2736)

	helper.SendTree(tree)

	logger.Info("ClusterInit.End", "Success!\nKonsultant => "+konsultantDriver.GetBaseDriver().IPAddress+"\nKonduktor => "+konduktorDriver.GetBaseDriver().IPAddress+"\nKlerk => "+klerkDriver.GetBaseDriver().IPAddress)

}

func (c *Cluster) Save() {

	savePath := path.Join(c.DeployDir, "data.mk")
	file, err := os.Create(savePath)
	defer file.Close()
	if err == nil {
		var buffer bytes.Buffer
		buffer.WriteString(c.Driver.DriverName + "\n")
		buffer.WriteString(c.Driver.Config["access-token"] + "\n")
		file.Write(buffer.Bytes())
	}

	for _, p := range c.Partikles {
		p.Save()
	}

	fmt.Printf("%+v\n", c)
}

func savePublicPEMKey(fileName string, pubkey rsa.PublicKey) string {
	pub, err := ssh.NewPublicKey(&pubkey)
	utils.PrintExitIfError(err, "Cannot generate SSH public keys", 3)
	keyBytes := ssh.MarshalAuthorizedKey(pub)
	err = ioutil.WriteFile(fileName, keyBytes, 0655)
	utils.PrintExitIfError(err, "Cannot save SSH public keys", 4)
	return string(keyBytes)
}

// func getIPAddr(b *vagrant.Box) *string {
// 	err := b.SSHConfig()
// 	utils.PrintExitIfError(err, "Cannot parse konsultant SSH Config", 2)

// 	fmt.Println("SSH Config Ok... Connecting...")

// 	connection, err := b.SSHConnect()
// 	utils.PrintExitIfError(err, "", 2)
// 	defer connection.Close()

// 	session, err := b.SSHSession(connection)
// 	defer session.Close()

// 	err = b.PrepareSSHShell(session)
// 	if err != nil {
// 		session.Close()
// 		connection.Close()
// 		fmt.Fprintf(os.Stderr, "%s", err.Error())
// 		os.Exit(2)
// 	}
// 	b.SSHIn <- "ip addr show"
// 	outIPShow := <-b.SSHOut

// 	b.SSHIn <- "exit"
// 	session.Wait()

// 	interfaces := parsers.ParseIPAddrShow(outIPShow)
// 	privAddr := virtualbox.GetPrivateAddress(interfaces)
// 	if privAddr == nil {
// 		fmt.Fprintf(os.Stderr, "Cannot get konsultant private IP...\r\n")
// 		os.Exit(2)
// 	} else {
// 		return privAddr
// 	}
// 	return nil
// }

// func gitCloneImages() (*vagrant.Box, *vagrant.Box, *vagrant.Box, error) {

// 	_, errKonsultant := git.PlainClone("/tmp/konsultant", false, &git.CloneOptions{
// 		URL: "http://nsurleraux.be:8081/mikrodock/vagrant-konsultant.git",
// 	})

// 	if errKonsultant != nil {
// 		return nil, nil, nil, errors.New("An error happened while cloning : " + errKonsultant.Error())
// 	}

// 	konsultant := &vagrant.Box{Path: "/tmp/konsultant", Name: "konsultant-vm"}

// 	_, errKonduktor := git.PlainClone("/tmp/konduktor", false, &git.CloneOptions{
// 		URL: "http://nsurleraux.be:8081/mikrodock/vagrant-konduktor.git",
// 	})

// 	if errKonduktor != nil {
// 		return nil, nil, nil, errors.New("An error happened while cloning : " + errKonduktor.Error())
// 	}

// 	konduktor := &vagrant.Box{Path: "/tmp/konduktor", Name: "konduktor-vm"}

// 	_, errKlerk := git.PlainClone("/tmp/klerk", false, &git.CloneOptions{
// 		URL: "http://nsurleraux.be:8081/mikrodock/vagrant-klerk.git",
// 	})

// 	if errKlerk != nil {
// 		return nil, nil, nil, errors.New("An error happened while cloning : " + errKlerk.Error())
// 	}

// 	klerk := &vagrant.Box{Path: "/tmp/klerk"}

// 	return konsultant, konduktor, klerk, nil

// }
