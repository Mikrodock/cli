package cluster

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"mikrodock-cli/digitalocean"
	"mikrodock-cli/generators"
	"mikrodock-cli/utils"
	"os"
	"path"
	"time"

	"golang.org/x/crypto/ssh"
)

type SSHConfig struct {
	PubKeyPath  string
	PrivKeyPath string
}

type Cluster struct {
	Name      string `json:"name,omitempty"`
	DeployDir string `json:"deployDir,omitempty"`
	SSHConfig SSHConfig
}

func NewCluster(name string, deployDir string) Cluster {
	clusterPath := path.Join(deployDir, name)
	return Cluster{Name: name, DeployDir: clusterPath}
}

func (c *Cluster) BoxPath(name string) string {
	return path.Join(c.DeployDir, "boxes", name)
}

func (c *Cluster) ClientCertPath() string {
	return path.Join(c.DeployDir, "client", "certs")
}

func (c *Cluster) ServerCertPath(name string) string {
	return path.Join(c.DeployDir, "boxes", name, "certs")
}

func (c *Cluster) Init() {
	errClusterDir := os.MkdirAll(c.DeployDir, os.FileMode(int(0733)))
	if errClusterDir != nil {
		fmt.Fprintf(os.Stderr, "Cannot create cluster directory ! %s\r\n", errClusterDir)
		os.Exit(4)
	}

	errKonsultantDir := os.MkdirAll(c.BoxPath("konsultant"), os.FileMode(int(0755)))
	errKonduktorDir := os.MkdirAll(c.BoxPath("konduktor"), os.FileMode(int(0755)))
	errKlerkDir := os.MkdirAll(c.BoxPath("klerk"), os.FileMode(int(0755)))

	if errKonsultantDir != nil || errKonduktorDir != nil || errKlerkDir != nil {
		fmt.Fprintf(os.Stderr, "Cannot create boxes directory ! \r\n %s %s %s\r\n", errKonduktorDir, errKonduktorDir, errKlerkDir)
		os.Exit(4)
	}

	fmt.Println("Generating Client Certs...")

	errCreateCertDir := os.MkdirAll(c.ClientCertPath(), os.FileMode(int(0755)))
	utils.PrintExitIfError(errCreateCertDir, "Cannot generate Cert Dir", 4)
	errCert := generators.GenerateClientCerts(c.Name+"-org", c.ClientCertPath())
	utils.PrintExitIfError(errCert, "", 2)

	fmt.Println("Generating SSH keys...")

	c.SSHConfig.PrivKeyPath = path.Join(c.DeployDir, "ssh_private_key")
	c.SSHConfig.PubKeyPath = path.Join(c.DeployDir, "ssh_public_key")

	reader := rand.Reader
	bitSize := 2048
	key, err := rsa.GenerateKey(reader, bitSize)
	utils.PrintExitIfError(err, "Cannot generate SSH keys", 3)
	publicKey := key.PublicKey

	savePEMKey(c.SSHConfig.PrivKeyPath, key)
	pubKey := savePublicPEMKey(c.SSHConfig.PubKeyPath, publicKey)

	keyGodo, err := digitalocean.AddSSHKey(c.Name+"-cluster", pubKey)
	utils.PrintExitIfError(err, "Cannot upload key", 5)

	konsDrop, err := digitalocean.CreateDockerDroplet("konsultant", []string{"konsultant"}, 1, keyGodo)
	utils.PrintExitIfError(err, "Cannot create konsultant droplet", 5)
	fmt.Println("Waiting Konsultant setup...")
	for konsDrop.Status != "active" {
		err = digitalocean.Refresh(konsDrop)
		utils.PrintExitIfError(err, "Cannot refresh konsultant state", 5)
		time.Sleep(5 * time.Second)
	}

	fmt.Printf("Status is now %s ! Waiting 10 seconds to avoid any conflict...\r\n", konsDrop.Status)
	time.Sleep(10 * time.Second)

	ipKonsultant, err := konsDrop.PrivateIPv4()
	utils.PrintExitIfError(err, "Cannot get konsultant private IP", 5)
	fmt.Printf("Konsultant private IP : %s\r\n", ipKonsultant)

	fmt.Printf("Generating Server certs...\r\n")
	errDir := os.MkdirAll(c.ServerCertPath("konsultant"), os.FileMode(int(0755)))
	utils.PrintExitIfError(errDir, "Cannot create Box Cert dir", 4)
	generators.GenerateServerCerts(c.Name+"-konsultant-org", konsDrop, c.ClientCertPath(), c.ServerCertPath("konsultant"), true)

	err = digitalocean.StartSSHConnection(konsDrop, c.SSHConfig.PrivKeyPath)
	utils.PrintExitIfError(err, "Cannot start SSH Connection", 6)

	fmt.Println("Copying certs...")

	_, err = digitalocean.SSHCommand(konsDrop, "mkdir -p /etc/docker/")
	utils.PrintExitIfError(err, "Cannot exec SSH command", 6)
	err = digitalocean.CopyPath(path.Join(c.ServerCertPath("konsultant"), "server-key.pem"), "/etc/docker", konsDrop)
	utils.PrintExitIfError(err, "Cannot SCP file", 6)
	err = digitalocean.CopyPath(path.Join(c.ServerCertPath("konsultant"), "server-cert.pem"), "/etc/docker", konsDrop)
	utils.PrintExitIfError(err, "Cannot SCP file", 6)
	err = digitalocean.CopyPath(path.Join(c.ClientCertPath(), "ca.pem"), "/etc/docker", konsDrop)
	utils.PrintExitIfError(err, "Cannot SCP file", 6)

	fmt.Println("Stopping docker")

	_, err = digitalocean.SSHCommand(konsDrop, "service docker stop")
	utils.PrintExitIfError(err, "Cannot stop docker service", 6)

	_, err = digitalocean.SSHCommand(konsDrop, `if [ ! -z "$(ip link show docker0)" ]; then sudo ip link delete docker0; fi`)
	utils.PrintExitIfError(err, "Cannot delete docker0 interface", 6)

	fmt.Println("Updating docker conf")

	_, err = digitalocean.SSHCommand(konsDrop, `sed -i 's;DOCKER_OPTS=.*;DOCKER_OPTS="--dns 8.8.8.8 --dns=8.8.4.4 --tlsverify --tlscacert=etc/docker/ca.pem --tlscert=etc/docker/server-cert.pem --tlskey=etc/docker/server-key.pem -H=0.0.0.0:2376";' /etc/default/docker`)
	utils.PrintExitIfError(err, "Cannot edit docker conf", 6)

	fmt.Println("Starting docker")

	_, err = digitalocean.SSHCommand(konsDrop, `service docker start`)
	utils.PrintExitIfError(err, "Cannot start docker service", 6)

	// res, err := digitalocean.SSHCommand(konsDrop, "ip addr show")
	// utils.PrintExitIfError(err, "Cannot exec SSH command", 6)
	// fmt.Println(res)

	// res, err = digitalocean.SSHCommand(konsDrop, "cat /etc/hosts")
	// utils.PrintExitIfError(err, "Cannot exec SSH command", 6)
	// fmt.Println(res)

	// digitalocean.CloseSSH(konsDrop)

	// err = digitalocean.DeleteDroplet(konsDrop)
	// utils.PrintExitIfError(err, "Cannot delete konsultant droplet", 5)

}

func savePEMKey(fileName string, key *rsa.PrivateKey) {
	outFile, err := os.Create(fileName)
	utils.PrintExitIfError(err, "Cannot create file", 4)
	defer outFile.Close()

	var privateKey = &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}

	err = pem.Encode(outFile, privateKey)
	utils.PrintExitIfError(err, "Cannot encode key", 3)
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
