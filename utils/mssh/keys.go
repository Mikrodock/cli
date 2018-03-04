package mssh

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
	"os"

	"golang.org/x/crypto/ssh"
)

func LoadPrivateKey(file string) (ssh.Signer, error) {
	buf, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	key, err := ssh.ParsePrivateKey(buf)
	if err != nil {
		return nil, err
	}

	return key, nil
}

func ComputePublicFingerprint(privKey ssh.Signer) string {
	return ssh.FingerprintLegacyMD5(privKey.PublicKey())
}

func CreatePrivateKey(privKeyFile string) error {
	// We only need to save the private key because the public key can be extracted from there
	reader := rand.Reader
	bitSize := 2048
	key, err := rsa.GenerateKey(reader, bitSize)
	if err != nil {
		return err
	}

	outFile, err := os.Create(privKeyFile)
	outFile.Chmod(os.FileMode(0600))
	if err != nil {
		return err
	}
	defer outFile.Close()

	var privateKey = &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}

	err = pem.Encode(outFile, privateKey)
	if err != nil {
		return err
	}

	return nil
}
