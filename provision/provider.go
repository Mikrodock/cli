package provision

import (
	"errors"
	"fmt"
	"mikrodock-cli/drivers"
	"mikrodock-cli/utils"
	"net/http"
	"path/filepath"

	dockerClientAPI "github.com/docker/docker/client"

	"github.com/docker/go-connections/tlsconfig"
)

type Provider interface {
	MatchOS(osType utils.OSType) bool
	DetectDocker() (bool, error)
	InstallDocker() error
	InstallPackage(pkgName string) error
	ConfigureDocker(additionnalConfig string) error
	StartDocker() error
	StopDocker() error
	CreateDirectory(path string) error
	SetDriver(driver drivers.Driver)
	GetBaseProvider() *BaseProvider
}

type BaseProvider struct {
	Driver drivers.Driver
}

func (pb *BaseProvider) MatchOS(osType utils.OSType) bool {
	return false
}

func (pb *BaseProvider) CreateDirectory(path string) error {
	return errors.New("BaseProvider cannot be used")
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

func (pb *BaseProvider) ConnectDocker(partikleCertPath string, dockerCertPath string) (*dockerClientAPI.Client, error) {
	var client *http.Client
	options := tlsconfig.Options{
		CAFile:             filepath.Join(dockerCertPath, "ca.cert"),
		CertFile:           filepath.Join(partikleCertPath, "cert.pem"),
		KeyFile:            filepath.Join(partikleCertPath, "key.pem"),
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
	}

	headers := make(map[string]string)

	if pb.Driver == nil {
		fmt.Println("Driver == nil!!!")
	}

	driverURL := pb.Driver.GetDockerURL()

	cli, err := dockerClientAPI.NewClient(driverURL, "1.27", client, headers)
	if err != nil {
		return cli, err
	}
	return cli, nil
}
