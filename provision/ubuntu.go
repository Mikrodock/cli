package provision

import (
	"fmt"
	"mikrodock-cli/drivers"
	"mikrodock-cli/utils"
	"os/exec"
	"strings"
)

type UbuntuProvider struct {
	BaseProvider
	Driver drivers.Driver
}

func (up *UbuntuProvider) MatchOS(osType utils.OSType) bool {
	return osType == utils.Ubuntu
}

func (up *UbuntuProvider) DetectDocker() (bool, error) {
	_, stderr, err := up.Driver.SSHCommand("docker version --format '{{json .}}'")

	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			if strings.Contains(stderr, "not found") {
				return false, nil
			}
		}
		return false, err
	}

	return true, nil
}

func (up *UbuntuProvider) InstallDocker() error {
	_, _, err := up.Driver.SSHCommand("apt-get update &&& apt-get -y install docker")
	return err
}

func (up *UbuntuProvider) InstallPackage(pkgName string) error {
	_, _, err := up.Driver.SSHCommand("apt-get update && apt-get -y install powerdns")
	return err
}

func (up *UbuntuProvider) ConfigureDocker(additionnalConfig string) error {

	fmt.Printf("Additionnal config : %s \n", additionnalConfig)

	// TODO : Add labels, registries, flags and env

	/* {{ range .EngineOptions.Labels }}--label {{.}}
	{{ end }}{{ range .EngineOptions.InsecureRegistry }}--insecure-registry {{.}}
	{{ end }}{{ range .EngineOptions.RegistryMirror }}--registry-mirror {{.}}
	{{ end }}{{ range .EngineOptions.ArbitraryFlags }}--{{.}}
	{{ end }}
	'
	{{range .EngineOptions.Env}}export \"{{ printf "%q" . }}\"
	{{end}} */

	dockerConf := `DOCKER_OPTS='
-H tcp://0.0.0.0:2376
-H unix:///var/run/docker.sock
--tlsverify
--tlscacert /etc/docker/ca.cert
--tlscert /etc/docker/cert.pem
--tlskey /etc/docker/key.pem
` + additionnalConfig + `'`

	fmt.Printf("Total config : %s \n", dockerConf)

	confCmd := fmt.Sprintf("printf %%s \"%s\" | tee /etc/default/docker", dockerConf)

	_, _, err := up.Driver.SSHCommand(confCmd)

	return err
}

func (up *UbuntuProvider) StartDocker() error {
	_, _, err := up.Driver.SSHCommand("service docker start")
	return err
}

func (up *UbuntuProvider) StopDocker() error {
	_, _, err := up.Driver.SSHCommand("service docker stop")
	return err
}

func (up *UbuntuProvider) GetBaseProvider() *BaseProvider {
	return &up.BaseProvider
}

func (up *UbuntuProvider) SetDriver(d drivers.Driver) {
	up.Driver = d
	up.BaseProvider.Driver = d
}

func (up *UbuntuProvider) CreateDirectory(path string) error {
	_, _, err := up.Driver.SSHCommand("mkdir -p " + path)
	return err
}
