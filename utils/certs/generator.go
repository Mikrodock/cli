package certs

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"time"
)

type CertOpts struct {
	CertFile, KeyFile, CAFile, CAKeyFile, Organization string
	MainHost                                           string
	AliasHosts                                         []string
	AliasIPs                                           []string
	KeyBits                                            int
	MasterMode                                         bool
}

type CertGenerator interface {
	GenerateCACert(certFile, keyFile, organization string, keyBits int) error
	GenerateCert(opts *CertOpts) error
}

type X509CertGenerator struct{}

func NewX509CertGenerator() *X509CertGenerator {
	return &X509CertGenerator{}
}

func (xGen *X509CertGenerator) GenerateCACert(certFile string, keyFile string, organization string, keyBits int) error {
	template, err := xGen.newCertificate(organization)
	if err != nil {
		return err
	}

	template.IsCA = true
	template.KeyUsage |= x509.KeyUsageCertSign
	template.KeyUsage |= x509.KeyUsageKeyEncipherment
	template.KeyUsage |= x509.KeyUsageKeyAgreement

	priv, err := rsa.GenerateKey(rand.Reader, keyBits)
	if err != nil {
		return err
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	if err != nil {
		return err
	}

	certOut, err := os.Create(certFile)
	if err != nil {
		return err
	}

	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	certOut.Close()

	keyOut, err := os.OpenFile(keyFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err

	}

	pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	keyOut.Close()

	return nil
}

func (xGen *X509CertGenerator) GenerateCert(opts *CertOpts) error {
	template, err := xGen.newCertificate(opts.Organization)
	if err != nil {
		return err
	}

	// Server certificate
	if len(opts.AliasIPs) != 1 || opts.AliasIPs[0] != "" || len(opts.AliasHosts) != 1 || opts.AliasHosts[0] != "" {
		if len(opts.AliasHosts) > 1 && opts.AliasHosts[0] != "" {
			for _, h := range opts.AliasHosts {
				template.DNSNames = append(template.DNSNames, h)
			}
		}

		if len(opts.AliasIPs) > 1 && opts.AliasIPs[0] != "" {
			for _, h := range opts.AliasIPs {
				template.IPAddresses = append(template.IPAddresses, net.ParseIP(h))
			}
		}

		template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}

		if opts.MasterMode {
			template.ExtKeyUsage = append(template.ExtKeyUsage, x509.ExtKeyUsageClientAuth)
		}

	} else {
		template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
		template.KeyUsage = x509.KeyUsageDigitalSignature
	}

	if opts.MasterMode {
		template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
		template.ExtKeyUsage = append(template.ExtKeyUsage, x509.ExtKeyUsageClientAuth)

	} else {
		template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	}
	// client

	tlsCert, err := tls.LoadX509KeyPair(opts.CAFile, opts.CAKeyFile)
	if err != nil {
		return err
	}

	priv, err := rsa.GenerateKey(rand.Reader, opts.KeyBits)
	if err != nil {
		return err
	}

	x509Cert, err := x509.ParseCertificate(tlsCert.Certificate[0])
	if err != nil {
		return err
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, template, x509Cert, &priv.PublicKey, tlsCert.PrivateKey)
	if err != nil {
		return err
	}

	certOut, err := os.Create(opts.CertFile)
	if err != nil {
		return err
	}

	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	certOut.Close()

	keyOut, err := os.OpenFile(opts.KeyFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}

	pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	keyOut.Close()

	return nil
}

func (xGen *X509CertGenerator) newCertificate(org string) (*x509.Certificate, error) {
	now := time.Now()
	// need to set notBefore slightly in the past to account for time
	// skew in the VMs otherwise the certs sometimes are not yet valid
	notBefore := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute()-5, 0, 0, time.Local)
	notAfter := notBefore.Add(time.Hour * 24 * 1080)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, err
	}

	return &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{org},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageKeyAgreement,
		BasicConstraintsValid: true,
	}, nil

}
