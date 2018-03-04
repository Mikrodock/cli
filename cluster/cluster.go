package cluster

import (
	"crypto/rsa"
	"fmt"
	"io/ioutil"
	"mikrodock-cli/drivers"
	"mikrodock-cli/logger"
	"mikrodock-cli/utils"
	"path"

	"golang.org/x/crypto/ssh"
)

type ClusterDriver struct {
	DriverName string
	Config     map[string]string
}

type Cluster struct {
	Name      string `json:"name,omitempty"`
	DeployDir string `json:"deployDir,omitempty"`
	Driver    ClusterDriver
}

func (c *Cluster) Init() {

	generateMinimalFiles(c)

	logger.Info("ClusterInit", "Creating Konsultant Machine")
	initDriver, err := drivers.NewDriver(c.Driver.DriverName, c.Driver.Config)
	if err != nil {
		logger.Fatal("ClusterInit", err.Error())
	}

	driverConfig := make(map[string]interface{})
	driverConfig["ssh-key-path"] = path.Join(c.SSHPath(), "private_key")
	driverConfig["name"] = "konsultant"
	// TODO : EXTENDS

	konsultantDriver, err := initDriver(driverConfig)
	logger.Info("ClusterInit.Konsultant", "PreCreate OK")
	if err != nil {
		logger.Fatal("ClusterInit.Konsultant", err.Error())
	}
	if err = konsultantDriver.Create(); err != nil {
		logger.Fatal("ClusterInit.Konsultant", err.Error())
	}

	logger.Info("ClusterInit.Konsultant", "Konsultant Machine Created")

	provider := getProvider(konsultantDriver)

	logger.Info("ClusterInit.Security", "Generating certificates...")

	makeCerts(c, konsultantDriver)
	configureDocker(provider)

	logger.Debug("Cluster.Konsultant.Docker", fmt.Sprintf("docker --tlsverify --tlscacert=%s --tlscert=%s --tlskey=%s -H=%s:2376 version\r\n", path.Join(c.DockerConfigPath(), "ca.cert"), path.Join(c.DockerConfigPath(), "cert.pem"), path.Join(c.DockerConfigPath(), "key.pem"), konsultantDriver.GetBaseDriver().IPAddress))
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
