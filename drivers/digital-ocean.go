package drivers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"mikrodock-cli/logger"
	mSSSH "mikrodock-cli/utils/mssh"
	"time"

	"github.com/tmc/scp"

	"golang.org/x/crypto/ssh"
	"golang.org/x/oauth2"

	"github.com/digitalocean/godo"
)

const (
	DefaultRegion string = "lon1"
	DefaultSize   string = "1gb"
)

type DigitalOceanDriver struct {
	BaseDriver
	AccessToken string
	DropletID   int
	Fingerprint string

	sshClient *ssh.Client
}

func (d *DigitalOceanDriver) PreCreate(conf map[string]interface{}) error {

	// Upload the public key
	client, err := d.getClient()
	if err != nil {
		return err
	}

	if conf["ssh-key-path"] == nil {
		return errors.New("No SSH key provided")
	}
	pKey, err := mSSSH.LoadPrivateKey(conf["ssh-key-path"].(string))
	if err != nil {
		return err
	}
	fingerprint := mSSSH.ComputePublicFingerprint(pKey)
	key, resp, err := client.Keys.GetByFingerprint(context.TODO(), fingerprint)
	if err != nil && resp.StatusCode != 404 {
		return err
	}
	if conf["name"] == nil {
		return errors.New("No name provided")
	}

	d.MachineName = conf["name"].(string)
	d.RawConfig = conf

	if key == nil {
		logger.Info("Driver.DigitalOcean", "Uploading new SSH key")
		pub := pKey.PublicKey()
		keyBytes := ssh.MarshalAuthorizedKey(pub)
		pubString := string(keyBytes)
		// We need to upload the key first
		request := &godo.KeyCreateRequest{
			Name:      d.MachineName,
			PublicKey: pubString,
		}
		newKey, _, err := client.Keys.Create(context.TODO(), request)
		if err != nil {
			return err
		}
		d.Fingerprint = newKey.Fingerprint
	} else {
		d.Fingerprint = key.Fingerprint
	}

	d.SSHKeyPath = conf["ssh-key-path"].(string)
	d.sshClient = nil

	return nil
}

func (d *DigitalOceanDriver) Create() error {

	client, err := d.getClient()
	if err != nil {
		return err
	}

	var name, region, size string

	name = d.RawConfig["name"].(string)

	if d.RawConfig["region"] == nil {
		region = DefaultRegion
	} else {
		region = d.RawConfig["region"].(string)
	}

	if d.RawConfig["size"] == nil {
		size = DefaultSize
	} else {
		size = d.RawConfig["size"].(string)
	}

	if d.RawConfig["ssh-key-path"] == nil {
		return errors.New("No SSH key provided")
	}

	createRequest := godo.DropletCreateRequest{
		Name:   name,
		Size:   size,
		Region: region,
		Image: godo.DropletCreateImage{
			Slug: "docker",
		},
		SSHKeys: []godo.DropletCreateSSHKey{godo.DropletCreateSSHKey{
			Fingerprint: d.Fingerprint,
		}},
	}

	fmt.Printf("%#v\r\n", createRequest)

	ctx := context.TODO()

	drop, _, err := client.Droplets.Create(ctx, &createRequest)
	if err != nil {
		return err
	}

	d.DropletID = drop.ID

	okState, err := d.WaitState(Running, 20)
	if err != nil {
		return err
	}

	if !okState {
		logger.Warn("DigitalOcean.Driver", "WaitState timeout expired, assuming Droplet is Running")
	}
	logger.Info("DigitalOcean.Driver", "Waiting 10 seconds for final boot...")
	time.Sleep(10 * time.Second)

	drop, _, _ = client.Droplets.Get(context.TODO(), d.DropletID)

	pub, err := drop.PublicIPv4()
	if err != nil {
		return err
	}
	logger.Info("Driver.DigitalOcean", "The public IPv4 is "+pub)

	d.IPAddress = pub
	d.SSHPort = "22"
	d.SSHUser = "root"

	return nil
}

func (d *DigitalOceanDriver) DriverName() string {
	return "digital-ocean"
}

func (d *DigitalOceanDriver) GetDockerURL() string {
	return "tcp://" + d.IPAddress + ":2376"
}

func (d *DigitalOceanDriver) GetState() (State, error) {
	drop, err := d.getDroplet()
	if err != nil {
		return Unknown, fmt.Errorf("Cannot get Droplet : %s", err)
	}
	switch drop.Status {
	case "new":
		return InCreation, nil
	case "active":
		return Running, nil
	case "off":
		return Stopped, nil
	default:
		return Unknown, nil
	}
}

func (d *DigitalOceanDriver) WaitState(state State, timeout int) (bool, error) {
	currentState, err := d.GetState()
	timeoutCounter := 0
	if err != nil {
		return false, fmt.Errorf("Cannot get Droplet state : %s", err)
	}
	for currentState != state {
		time.Sleep(1 * time.Second)
		timeoutCounter++
		if timeoutCounter > timeout {
			return false, nil
		}
		currentState, err = d.GetState()
		if err != nil {
			return false, fmt.Errorf("Cannot get Droplet state : %s", err)
		}
	}
	return true, nil
}

func (d *DigitalOceanDriver) sshConnect() error {
	sshConfig := &ssh.ClientConfig{
		User: d.SSHUser,
		Auth: []ssh.AuthMethod{
			mSSSH.PublicKeyFile(d.SSHKeyPath),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	logger.Info("SSHUtils", "Opening connection to "+d.IPAddress+":"+d.SSHPort)
	retryCount := 1
	var _err error
	for retryCount < 3 {
		var err error
		d.sshClient, err = ssh.Dial("tcp", d.IPAddress+":"+d.SSHPort, sshConfig)
		if err != nil {
			logger.Warn("SSHUtils", "Cannot connect SSH to host, retrying in 10 seconds...")
			time.Sleep(10 * time.Second)
			retryCount++
			_err = err
		} else {
			_err = nil
			break
		}
	}

	if _err != nil {
		logger.Info("SSHUtils", "Connection open")
	}

	return _err
}

func (d *DigitalOceanDriver) CopyFile(source string, destination string) error {
	if d.sshClient == nil {
		err := d.sshConnect()
		if err != nil {
			return err
		}
	}
	sess, err := d.sshClient.NewSession()
	if err != nil {
		return err
	}
	defer sess.Close()

	return scp.CopyPath(source, destination, sess)
}

func (d *DigitalOceanDriver) SSHCommand(cmd string) (string, string, error) {
	if d.sshClient == nil {
		err := d.sshConnect()
		if err != nil {
			return "", "", err
		}
	}
	sess, err := d.sshClient.NewSession()
	if err != nil {
		return "", "", err
	}
	defer sess.Close()
	var stdoutBuf bytes.Buffer
	sess.Stdout = &stdoutBuf
	var stderrBuf bytes.Buffer
	sess.Stderr = &stderrBuf

	err = sess.Run(cmd)

	return stdoutBuf.String(), stderrBuf.String(), err
}

func (d *DigitalOceanDriver) Kill() error {
	panic("not implemented")
}

func (d *DigitalOceanDriver) Destroy() error {
	panic("not implemented")
}

func (d *DigitalOceanDriver) Start() error {
	panic("not implemented")
}

func (d *DigitalOceanDriver) Stop() error {
	panic("not implemented")
}

func (d *DigitalOceanDriver) Restart() error {
	panic("not implemented")
}

func (d *DigitalOceanDriver) GetBaseDriver() *BaseDriver {
	return &d.BaseDriver
}

// Provide some helpers functions

func (d *DigitalOceanDriver) getClient() (*godo.Client, error) {
	tSource := oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: d.AccessToken,
	})
	oauthClient := oauth2.NewClient(context.Background(), tSource)
	client := godo.NewClient(oauthClient)

	return client, nil
}

func (d *DigitalOceanDriver) getDroplet() (*godo.Droplet, error) {
	client, _ := d.getClient()
	drop, _, err := client.Droplets.Get(context.TODO(), d.DropletID)
	return drop, err
}
