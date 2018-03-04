package provision

import (
	"errors"
	"mikrodock-cli/drivers"
	"mikrodock-cli/utils"
	"net/http"
	"path/filepath"

	dockerClient "github.com/docker/docker/client"

	"github.com/docker/go-connections/tlsconfig"
)

type Provider interface {
	MatchOS(osType utils.OSType) bool
	DetectDocker() (bool, error)
	InstallDocker() error
	ConfigureDocker() error
	StartDocker() error
	StopDocker() error
	SetDriver(driver drivers.Driver)
	GetBaseProvider() *BaseProvider
}

type BaseProvider struct {
	Driver drivers.Driver
}

func (pb *BaseProvider) MatchOS(osType utils.OSType) bool {
	return false
}

func (pb *BaseProvider) DetectDocker() (bool, error) {
	return false, errors.New("BaseProvider cannot be used")
}

func (pb *BaseProvider) InstallDocker() error {
	return errors.New("BaseProvider cannot be used")
}

func (pb *BaseProvider) ConfigureDocker() error {
	return errors.New("BaseProvider cannot be used")
}

func (pb *BaseProvider) StartDocker() error {
	return errors.New("BaseProvider cannot be used")
}

func (pb *BaseProvider) StopDocker() error {
	return errors.New("BaseProvider cannot be used")
}

func (pb *BaseProvider) SetDriver(driver drivers.Driver) {
	pb.Driver = driver
}

func (pb *BaseProvider) GetBaseProvider() *BaseProvider {
	return pb
}

func (pb *BaseProvider) ConnectDocker(dockerCertPath string) (*dockerClient.Client, error) {
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

	cli, err := dockerClient.NewClient(pb.Driver.GetDockerURL(), "1.27", client, nil)
	if err != nil {
		return cli, err
	}
	return cli, nil
}
