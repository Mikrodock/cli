package vagrant

import (
	"fmt"
	"os/exec"
)

type VagrantWorker struct {
	Command string
	Args    []string
	Dir     string
	Output  string
	Error   error
}

func (cmd *VagrantWorker) Run() {
	args := append([]string{cmd.Command}, cmd.Args...)
	cmdExec := exec.Command("vagrant", args...)
	cmdExec.Dir = cmd.Dir
	out, err := cmdExec.Output()
	if err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			fmt.Printf("%s\r\n", string(exiterr.Stderr))
		}
		cmd.Error = err
	}

	cmd.Output = string(out)
}
