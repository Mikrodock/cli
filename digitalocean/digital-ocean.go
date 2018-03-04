package digitalocean

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strconv"

	dockerClient "github.com/docker/docker/client"
	"github.com/docker/go-connections/tlsconfig"
	"github.com/tmc/scp"

	"github.com/digitalocean/godo/context"

	"github.com/digitalocean/godo"
	"golang.org/x/crypto/ssh"
	"golang.org/x/oauth2"
)

const (
	pat = "5b5aa919fc442f247d82fee6461c0294ff084dbe3cc4dd8fce735f87d18e24bf"
)

var client *godo.Client

var sshConnectionPerID map[int]*ssh.Client

type TokenSource struct {
	AccessToken string
}

func (t *TokenSource) Token() (*oauth2.Token, error) {
	token := &oauth2.Token{
		AccessToken: t.AccessToken,
	}
	return token, nil
}

func init() {
	tokenSource := &TokenSource{
		AccessToken: pat,
	}

	sshConnectionPerID = make(map[int]*ssh.Client)

	oauthClient := oauth2.NewClient(context.Background(), tokenSource)
	client = godo.NewClient(oauthClient)

}

func StartSSHConnection(droplet *godo.Droplet, sshKeyPath string) error {

	if sshConnectionPerID[droplet.ID] != nil {
		return nil
	}

	key, err := ioutil.ReadFile(sshKeyPath)
	if err != nil {
		return err
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return err
	}

	config := &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			// Use the PublicKeys method for remote authentication.
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	pubIP, err := droplet.PublicIPv4()
	if err != nil {
		return err
	}

	client, err := ssh.Dial("tcp", pubIP+":22", config)
	if err != nil {
		return err
	}

	sshConnectionPerID[droplet.ID] = client

	return nil
}

func CopyPath(origin string, dest string, droplet *godo.Droplet) error {
	if sshConnectionPerID[droplet.ID] == nil {
		return fmt.Errorf("No open SSH connection")
	}
	session, err := sshConnectionPerID[droplet.ID].NewSession()
	if err != nil {
		return err
	}
	defer session.Close()
	return scp.CopyPath(origin, dest, session)
}

func CloseSSH(droplet *godo.Droplet) {
	if sshConnectionPerID[droplet.ID] != nil {
		sshConnectionPerID[droplet.ID].Close()
		delete(sshConnectionPerID, droplet.ID)
	}
}

func SSHCommand(droplet *godo.Droplet, cmd string) (string, error) {
	if sshConnectionPerID[droplet.ID] == nil {
		return "", fmt.Errorf("No open SSH connection")
	}

	conn := sshConnectionPerID[droplet.ID]
	sess, err := conn.NewSession()
	if err != nil {
		return "", err
	}
	defer sess.Close()

	resBytes, err := sess.Output(cmd)
	if err != nil {
		return "", err
	}
	return string(resBytes), nil
}

func Refresh(droplet *godo.Droplet) error {
	ctx := context.TODO()
	drop, _, err := client.Droplets.Get(ctx, droplet.ID)
	if err != nil {
		return err
	}
	*droplet = *drop
	return nil
}

func AddSSHKey(name, key string) (*godo.Key, error) {
	createRequest := &godo.KeyCreateRequest{
		Name:      name,
		PublicKey: key,
	}
	ctx := context.TODO()
	keyGodo, _, err := client.Keys.Create(ctx, createRequest)

	if err != nil {
		return nil, err
	}
	return keyGodo, nil

}

func DeleteDroplet(droplet *godo.Droplet) error {
	ctx := context.TODO()
	_, err := client.Droplets.Delete(ctx, droplet.ID)
	return err
}

func CreateDockerDroplet(dropletName string, tags []string, gbSize int, keyGodo *godo.Key) (*godo.Droplet, error) {

	key := godo.DropletCreateSSHKey{Fingerprint: keyGodo.Fingerprint}

	fmt.Printf("DEBUG : Create %s, %s, %s\r\n", dropletName, strconv.Itoa(gbSize), keyGodo.Fingerprint)

	createRequest := &godo.DropletCreateRequest{
		Name:   dropletName,
		Region: "lon1",
		Size:   strconv.Itoa(gbSize) + "gb",
		Tags:   tags,
		Image: godo.DropletCreateImage{
			Slug: "docker",
		},
		PrivateNetworking: true,
		SSHKeys:           []godo.DropletCreateSSHKey{key},
	}

	ctx := context.TODO()

	newDroplet, _, err := client.Droplets.Create(ctx, createRequest)

	if err != nil {
		return nil, err
	}

	return newDroplet, nil
}

func ConnectDocker(dockerCertPath string, host string, version string) (*dockerClient.Client, error) {
	var client *http.Client
	options := tlsconfig.Options{
		CAFile:             filepath.Join(dockerCertPath, "ca.pem"),
		CertFile:           filepath.Join(dockerCertPath, "cert.pem"),
		KeyFile:            filepath.Join(dockerCertPath, "key.pem"),
		InsecureSkipVerify: false,
	}
	tlsc, err := tlsconfig.Client(options)
	if err != nil {
		return nil, err
	}

	client = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsc,
		},
		CheckRedirect: dockerClient.CheckRedirect,
	}

	cli, err := dockerClient.NewClient(host, version, client, nil)
	if err != nil {
		return cli, err
	}
	return cli, nil
}
