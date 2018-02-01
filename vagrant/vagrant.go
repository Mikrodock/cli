package vagrant

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/kevinburke/ssh_config"
	"golang.org/x/crypto/ssh"

	"github.com/termie/go-shutil"
)

type State string

const (
	Unknown    = "Unknown"
	NotCreated = "Not Created"
	Running    = "Running"
	Saved      = "Saved"
	PowerOff   = "Power off"
	Aborted    = "Aborted"
	Preparing  = "Preparing"
)

type Box struct {
	Path       string
	validState bool
	State      State
	SSHIn      chan<- string
	SSHOut     <-chan string
}

func (b *Box) Up() (bool, error) {
	state, _ := b.Status()
	if state != NotCreated && state != PowerOff {
		return false, fmt.Errorf("Cannot up VM, state is : %s", state)
	}
	w := VagrantWorker{Command: "up", Args: []string{}, Dir: b.Path}
	w.Run()
	if w.Error != nil {
		return false, w.Error
	}
	return true, nil
}

func (b *Box) Status() (State, error) {
	w := VagrantWorker{Command: "status", Args: []string{"--machine-readable"}, Dir: b.Path}
	w.Run()
	if w.Error != nil {
		return Unknown, w.Error
	}
	data := GetValue(w.Output, "state")
	state, _ := toState(data[0])
	return state, nil
}

func (b *Box) Destroy() (bool, error) {
	w := VagrantWorker{Command: "destroy", Args: []string{"-f", "--machine-readable"}, Dir: b.Path}
	w.Run()
	if w.Error != nil {
		return false, w.Error
	}
	return true, nil
}

func (b *Box) Clear() {
	if state, _ := b.Status(); state != NotCreated {
		b.Destroy()
	}
	os.RemoveAll(b.Path)
}

// TODO : edit .vagrant/...../synced_folder and vagrant_cwd to reflect the directory change and restart VM
func (b *Box) MoveClear(dst string) {
	// Copy (cause it's cross devices /tmp and /home not in the same partition) .vagrant and Vagrantfile, delete all files in old path and set new path
	fmt.Printf("Copying %s to %s\r\n", path.Join(b.Path, ".vagrant"), path.Join(dst, ".vagrant"))
	fmt.Printf("Copying %s to %s\r\n", path.Join(b.Path, "Vagrantfile"), path.Join(dst, "Vagrantfile"))
	err1 := shutil.CopyTree(path.Join(b.Path, ".vagrant"), path.Join(dst, ".vagrant"), nil)
	err2 := shutil.CopyFile(path.Join(b.Path, "Vagrantfile"), path.Join(dst, "Vagrantfile"), false)
	if err1 != nil {
		fmt.Printf("Error while copying .vagrant : %v\r\n", err1)
	}
	if err2 != nil {
		fmt.Printf("Error while copying Vagrantfile : %v\r\n", err2)
	}
	os.RemoveAll(b.Path)
	b.Path = dst
}

func (b *Box) SSHConfig() (*ssh_config.Config, error) {
	w := VagrantWorker{Command: "ssh-config", Args: []string{}, Dir: b.Path}
	w.Run()
	if w.Error != nil {
		return nil, errors.New("Cannot get ssh-config : " + w.Error.Error())
	}
	config, err := ssh_config.Decode(strings.NewReader(w.Output))
	if err != nil {
		return nil, errors.New("Cannot decode ssh-config : " + err.Error())
	}
	return config, nil
}

func (b *Box) SSHConnect(host string, port string, user string, identity string) (*ssh.Client, error) {
	key, err := ioutil.ReadFile(identity)
	if err != nil {
		return nil, fmt.Errorf("unable to read private key: %v", err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("unable to parse private key: %v", err)
	}

	config := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	client, err := ssh.Dial("tcp", host+":"+port, config)
	if err != nil {
		return nil, fmt.Errorf("unable to connect: %v", err)
	}
	return client, nil
}

func (b *Box) SSHSession(client *ssh.Client) (*ssh.Session, error) {
	session, err := client.NewSession()

	if err != nil {
		return nil, fmt.Errorf("unable to create session: %s", err)
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	if err := session.RequestPty("xterm", 80, 40, modes); err != nil {
		session.Close()
		return nil, fmt.Errorf("unable to create pty: %s", err)
	}

	return session, nil

}

func (b *Box) PrepareSSHShell(session *ssh.Session) error {
	w, err := session.StdinPipe()
	if err != nil {
		return err
	}
	r, err := session.StdoutPipe()
	if err != nil {
		return err
	}
	in, out := MuxShell(w, r)
	if err := session.Start("/bin/sh"); err != nil {
		return err
	}
	<-out //ignore the shell initial output
	b.SSHIn = in
	b.SSHOut = out

	return nil

}

func toState(state string) (State, error) {
	switch state {
	case "running":
		return Running, nil
	case "not_created":
		return NotCreated, nil
	case "saved":
		return Saved, nil
	case "poweroff":
		return PowerOff, nil
	case "aborted":
		return Aborted, nil
	case "preparing":
		return Preparing, nil
	default:
		return Unknown, fmt.Errorf("Unknown state: %s", state)
	}

}
