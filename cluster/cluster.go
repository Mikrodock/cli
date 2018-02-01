package cluster

import (
	"errors"
	"fmt"
	"mikrodock-cli/filters/virtualbox"
	"mikrodock-cli/parsers"
	"mikrodock-cli/vagrant"
	"os"
	"path"

	git "gopkg.in/src-d/go-git.v4"
)

type Cluster struct {
	Name      string `json:"name,omitempty"`
	DeployDir string `json:"name,omitempty"`
}

func NewCluster(name string, deployDir string) Cluster {
	clusterPath := path.Join(deployDir, name)
	return Cluster{name, clusterPath}
}

func (c *Cluster) BoxPath(name string) string {
	return path.Join(c.DeployDir, "boxes", name)
}

func (c *Cluster) Init() {
	errClusterDir := os.MkdirAll(c.DeployDir, os.FileMode(int(0733)))
	if errClusterDir != nil {
		fmt.Fprintf(os.Stderr, "Cannot create cluster directory ! %s\r\n", errClusterDir)
		os.Exit(4)
	}

	errKonsultantDir := os.MkdirAll(c.BoxPath("konsultant"), os.FileMode(int(0733)))
	errKonduktorDir := os.MkdirAll(c.BoxPath("konduktor"), os.FileMode(int(0733)))
	errKlerkDir := os.MkdirAll(c.BoxPath("klerk"), os.FileMode(int(0733)))

	if errKonsultantDir != nil || errKonduktorDir != nil || errKlerkDir != nil {
		fmt.Fprintf(os.Stderr, "Cannot create boxes directory ! \r\n %s %s %s\r\n", errKonduktorDir, errKonduktorDir, errKlerkDir)
		os.Exit(4)
	}

	fmt.Println("Cloning repositories...")
	konsultant, konduktor, klerk, err := gitCloneImages()

	if err != nil {
		fmt.Println(err.Error())
		os.Exit(2)
	}
	fmt.Println("Git repositories cloned... Starting Konsultant...")

	_, err = konsultant.Up()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot start Konsultant VM : %s", err.Error())
		os.Exit(2)
	}

	fmt.Println("Konsultant VM started, reading SSH configuration")
	sshconf, err := konsultant.SSHConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", err.Error())
		os.Exit(2)
	}
	host, err := sshconf.Get("konsultant-vm", "HostName")
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", err.Error())
		os.Exit(2)
	}
	user, err := sshconf.Get("konsultant-vm", "User")
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", err.Error())
		os.Exit(2)
	}
	port, err := sshconf.Get("konsultant-vm", "Port")
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", err.Error())
		os.Exit(2)
	}
	identity, err := sshconf.Get("konsultant-vm", "IdentityFile")
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", err.Error())
		os.Exit(2)
	}

	fmt.Println("SSH Config Ok... Connecting...")

	connection, err := konsultant.SSHConnect(host, port, user, identity)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", err.Error())
		os.Exit(2)
	}
	defer connection.Close()

	session, err := konsultant.SSHSession(connection)
	defer session.Close()

	err = konsultant.PrepareSSHShell(session)
	if err != nil {
		session.Close()
		connection.Close()
		fmt.Fprintf(os.Stderr, "%s", err.Error())
		os.Exit(2)
	}
	konsultant.SSHIn <- "ip addr show"
	outIPShow := <-konsultant.SSHOut

	konsultant.SSHIn <- "exit"
	session.Wait()

	konsultantInterfaces := parsers.ParseIPAddrShow(outIPShow)
	konsultantPrivAddr := virtualbox.GetPrivateAddress(konsultantInterfaces)
	if konsultantPrivAddr == nil {
		fmt.Fprintf(os.Stderr, "Cannot get konsultant private IP...\r\n")
		os.Exit(2)
	} else {
		fmt.Println("Konsultant private IP is " + *konsultantPrivAddr)
	}

	fmt.Println("Cleaning up...")
	konsultant.MoveClear(c.BoxPath("konsultant"))
	konduktor.Clear()
	klerk.Clear()

}

func gitCloneImages() (*vagrant.Box, *vagrant.Box, *vagrant.Box, error) {

	_, errKonsultant := git.PlainClone("/tmp/konsultant", false, &git.CloneOptions{
		URL: "http://nsurleraux.be:8081/mikrodock/vagrant-konsultant.git",
	})

	if errKonsultant != nil {
		return nil, nil, nil, errors.New("An error happened while cloning : " + errKonsultant.Error())
	}

	konsultant := &vagrant.Box{Path: "/tmp/konsultant"}

	_, errKonduktor := git.PlainClone("/tmp/konduktor", false, &git.CloneOptions{
		URL: "http://nsurleraux.be:8081/mikrodock/vagrant-konduktor.git",
	})

	if errKonduktor != nil {
		return nil, nil, nil, errors.New("An error happened while cloning : " + errKonduktor.Error())
	}

	konduktor := &vagrant.Box{Path: "/tmp/konduktor"}

	_, errKlerk := git.PlainClone("/tmp/klerk", false, &git.CloneOptions{
		URL: "http://nsurleraux.be:8081/mikrodock/vagrant-klerk.git",
	})

	if errKlerk != nil {
		return nil, nil, nil, errors.New("An error happened while cloning : " + errKlerk.Error())
	}

	klerk := &vagrant.Box{Path: "/tmp/klerk"}

	return konsultant, konduktor, klerk, nil

}
