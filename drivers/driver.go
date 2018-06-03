package drivers

import (
	"io"
	"os"

	"golang.org/x/crypto/ssh"
)

type SSHSettings struct {
	IPAddress string
	Port      string
	User      string
	KeyPath   string
}

// Driver is a way to create a host
// and contains info of a host
// The provider will perform the actions on the driver host
type Driver interface {
	PreCreate(conf map[string]interface{}) error

	// Create a host according to the driver flags.
	// It should populate the fields of the drivers when created
	Create() error

	DriverName() string

	// GetDockerURL returns the Docker Connection url
	GetDockerURL() string

	SSHShell() error

	GetState() (State, error)
	WaitState(state State, timeout int) (bool, error)

	GetBaseDriver() *BaseDriver
	SetBaseDriver(base BaseDriver)

	Kill() error
	Destroy() error
	Start() error
	Stop() error
	Restart() error

	SSHCommand(cmd string) (string, string, error)
	CopyFile(source string, destination string) error
	Copy(size int64, mode os.FileMode, fileName string, contents io.Reader, destinationPath string, session *ssh.Session) error
}
