package consul

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"time"
)

func bigIntHash(n *big.Int) []byte {
	h := sha1.New()
	h.Write(n.Bytes())
	return h.Sum(nil)
}

func GenerateCert(IPAddresses []string, CertFile, KeyFile, CACertFile, CAKeyFile string) error {

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	tpl := x509.Certificate{
		SubjectKeyId: bigIntHash(priv.N),
		IsCA:         false,
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		Subject: pkix.Name{
			Country:      []string{"BE"},
			CommonName:   "consul.local",
			Organization: []string{"MikroDock"},
		},
		SerialNumber: big.NewInt(365),
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(10, 0, 0),
	}

	for _, h := range IPAddresses {
		if ip := net.ParseIP(h); ip != nil {
			tpl.IPAddresses = append(tpl.IPAddresses, ip)
		}
	}

	tlsCert, err := tls.LoadX509KeyPair(CACertFile, CAKeyFile)
	if err != nil {
		return err
	}

	x509Cert, err := x509.ParseCertificate(tlsCert.Certificate[0])
	if err != nil {
		return err
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &tpl, x509Cert, &priv.PublicKey, tlsCert.PrivateKey)
	if err != nil {
		return err
	}

	certOut, err := os.Create(CertFile)
	if err != nil {
		return err
	}

	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	certOut.Close()

	keyOut, err := os.OpenFile(KeyFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}

	pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	keyOut.Close()

	return nil
}

func GenerateCACert(certFile, keyFile string) error {

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	caUsage := &x509.Certificate{
		SerialNumber: big.NewInt(1854),
		Subject: pkix.Name{
			Country:      []string{"BE"},
			CommonName:   "MikroConsulCA",
			Organization: []string{"MikroDock"},
		},
		SubjectKeyId: bigIntHash(priv.N),
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		IsCA:         true,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature | x509.KeyUsageCRLSign | x509.KeyUsageKeyEncipherment,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, caUsage, caUsage, &priv.PublicKey, priv)
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
