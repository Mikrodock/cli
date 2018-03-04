package mssh

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestCreatePrivateKey(t *testing.T) {
	err := CreatePrivateKey("/tmp/key.key")
	if err != nil {
		t.Errorf("Got an unexpected error while CreatePrivateKey : %s\r\n", err)
	}
	os.Mkdir("/tmp/noperm", os.FileMode(0400))
	err = CreatePrivateKey("/tmp/noperm/key.key")
	if err == nil {
		t.Errorf("Got no error while an Error was expected (noperm)")
	}
	os.Remove("/tmp/key.key")
}

func TestLoadPrivateKey(t *testing.T) {
	err := CreatePrivateKey("/tmp/key.key")
	if err != nil {
		t.Errorf("Got an unexpected error while CreatePrivateKey : %s\r\n", err)
	}
	defer os.Remove("/tmp/key.key")

	_, err = LoadPrivateKey("/tmp/key.key")
	if err != nil {
		t.Errorf("Got an unexpected error while LoadPrivateKey : %s\r\n", err)
	}

	_, err = LoadPrivateKey("/tmp/key.key2")

	if err == nil {
		t.Errorf("Got no error while an Error was expected (nofile)")
	}

	bigBuff := make([]byte, 750)
	ioutil.WriteFile("/tmp/badkey", bigBuff, 0666)
	defer os.Remove("/tmp/badkey")

	_, err = LoadPrivateKey("/tmp/badkey")

	if err == nil {
		t.Errorf("Got no error while an Error was expected (badkey)")
	}
}

func TestComputePublicFingerprint(t *testing.T) {
	err := CreatePrivateKey("/tmp/key.key")
	if err != nil {
		t.Errorf("Got an unexpected error while CreatePrivateKey : %s\r\n", err)
	}
	defer os.Remove("/tmp/key.key")

	signer, err := LoadPrivateKey("/tmp/key.key")
	if err != nil {
		t.Errorf("Got an unexpected error while LoadPrivateKey : %s\r\n", err)
	}
	ComputePublicFingerprint(signer)
}
