package drivers

import (
	"errors"
	"io"
	"os"

	"golang.org/x/crypto/ssh"
)

// BaseDriver is a common structure for drivers
type BaseDriver struct {
	IPAddress   string
	MachineID   int
	MachineName string
	SSHUser     string
	SSHPort     string
	SSHKeyPath  string
	RawConfig   map[string]interface{}
}

func (d *BaseDriver) Create() error {
	return errors.New("Base driver cannot create a host")
}

func (d *BaseDriver) DriverName() string {
	return "base (mock)"
}

func (d *BaseDriver) PreCreate(conf map[string]interface{}) error {
	return errors.New("Base driver cannot PreCreate")
}

func (d *BaseDriver) GetDockerURL() string {
	return "tcp://" + d.IPAddress + ":2376"
}

func (d *BaseDriver) GetState() (State, error) {
	return Unknown, errors.New("Base driver cannot get state")
}

func (d *BaseDriver) WaitState(state State, timeout int) (bool, error) {
	return false, errors.New("Base driver cannot wait state")
}

func (d *BaseDriver) Kill() error {
	return errors.New("Base driver cannot kill a host")
}

func (d *BaseDriver) Destroy() error {
	return errors.New("Base driver cannot destroy a host")
}

func (d *BaseDriver) Start() error {
	return errors.New("Base driver cannot start a host")
}

func (d *BaseDriver) Stop() error {
	return errors.New("Base driver cannot stop a host")
}

func (d *BaseDriver) Restart() error {
	return errors.New("Base driver cannot restart a host")
}

func (d *BaseDriver) GetBaseDriver() *BaseDriver {
	return d
}

func (d *BaseDriver) SSHCommand(cmd string) (string, string, error) {
	return "", "", errors.New("Base driver cannot exec SSH commands")
}

func (d *BaseDriver) SSHShell() error {
	return errors.New("Base driver cannot create SSH Shells")
}

func (d *BaseDriver) CopyFile(source string, destination string) error {
	return errors.New("Base driver cannot copy files")
}

func (d *BaseDriver) Copy(size int64, mode os.FileMode, fileName string, contents io.Reader, destinationPath string, session *ssh.Session) error {
	return errors.New("Base driver cannot copy files")
}
