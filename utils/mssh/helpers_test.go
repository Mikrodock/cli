package mssh

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestPublicKeyFile(t *testing.T) {
	err := CreatePrivateKey("/tmp/key.key")
	if err != nil {
		t.Errorf("Got an unexpected error while CreatePrivateKey : %s\r\n", err)
	}
	defer os.Remove("/tmp/key.key")

	auth := PublicKeyFile("/tmp/key.key")

	if auth == nil {
		t.Errorf("Got an unexpected error while PublicKeyFile\r\n")
	}

	auth = PublicKeyFile("/tmp/key2.key")

	if auth != nil {
		t.Errorf("Got an unexpected result while PublicKeyFile (nofile)\r\n")
	}

	bigBuff := make([]byte, 750)
	ioutil.WriteFile("/tmp/badkey", bigBuff, 0666)
	defer os.Remove("/tmp/badkey")

	auth = PublicKeyFile("/tmp/badkey")

	if auth != nil {
		t.Errorf("Got an unexpected result while PublicKeyFile (badkey)\r\n")
	}
}
