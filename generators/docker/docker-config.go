package docker

import (
	"fmt"
	"path"

	"github.com/digitalocean/godo"

	"github.com/docker/machine/libmachine/cert"
)

func GenerateClientCerts(orgName string, certDir string) error {
	generator := cert.NewX509CertGenerator()
	err := generator.GenerateCACertificate(path.Join(certDir, "ca.pem"), path.Join(certDir, "ca-key.pem"), orgName, 2048)
	if err != nil {
		return fmt.Errorf("Generating CA certificate failed: %s", err)
	}

	//Client Certs
	certOptions := &cert.Options{
		Hosts:       []string{""},
		CertFile:    path.Join(certDir, "cert.pem"),
		KeyFile:     path.Join(certDir, "key.pem"),
		CAFile:      path.Join(certDir, "ca.pem"),
		CAKeyFile:   path.Join(certDir, "ca-key.pem"),
		Org:         orgName,
		Bits:        2048,
		SwarmMaster: false,
	}

	err = cert.GenerateCert(certOptions)
	if err != nil {
		return fmt.Errorf("Generating Client certificate failed: %s", err)
	}

	return nil
}

func GenerateServerCerts(orgName string, drop *godo.Droplet, certDir string, boxCertDir string, isMaster bool) error {

	ip, _ := drop.PublicIPv4()
	privIP, _ := drop.PrivateIPv4()

	hosts := []string{"docker.local", ip, privIP, "localhost"}

	err := cert.GenerateCert(&cert.Options{
		Hosts:       hosts,
		CertFile:    path.Join(boxCertDir, "server-cert.pem"),
		KeyFile:     path.Join(boxCertDir, "server-key.pem"),
		CAFile:      path.Join(certDir, "ca.pem"),
		CAKeyFile:   path.Join(certDir, "ca-key.pem"),
		Org:         orgName,
		Bits:        2048,
		SwarmMaster: isMaster,
	})

	if err != nil {
		return err
	}
	return nil
}
