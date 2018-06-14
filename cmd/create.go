// Copyright © 2018 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"bytes"
	"encoding/json"
	"mikrodock-cli/cluster"
	"mikrodock-cli/logger"
	"net/http"
	"os"
	"path"
	"strconv"

	"github.com/spf13/cobra"
)

type PartikleConfig struct {
	Name      string
	IP        string
	SSHPort   int
	SSHUser   string
	MachineID int
	IsMaster  bool
}

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		c, err := cluster.LoadCluster(args[0])
		if err != nil {
			logger.Fatal("Node.Create", "Cannot load cluster : "+err.Error())
			return
		}

		for _, p := range c.Partikles {
			if p.Name() == "konduktor" {
				ip := p.IP()
				res, err := http.Post("http://"+ip+":10513/nodes", "application/json", bytes.NewBuffer([]byte{}))
				if err != nil {
					logger.Fatal("Node.Create.Post", err.Error())
				}
				if res.StatusCode == 201 {
					logger.Info("Kinetik.Service", "OK!")
					resJSON := &PartikleConfig{}
					json.NewDecoder(res.Body).Decode(resJSON)
					os.MkdirAll(c.PartiklePath(resJSON.Name), os.FileMode(0750))
					savePath := path.Join(c.PartiklePath(resJSON.Name), "data.mk")
					file, err := os.Create(savePath)
					defer file.Close()
					if err == nil {
						var buffer bytes.Buffer
						buffer.WriteString(resJSON.IP + "\n")
						buffer.WriteString(resJSON.Name + "\n")
						buffer.WriteString(c.Partikles[0].Driver.GetBaseDriver().SSHKeyPath + "\n")
						buffer.WriteString(strconv.Itoa(resJSON.SSHPort) + "\n")
						buffer.WriteString(resJSON.SSHUser + "\n")
						buffer.WriteString(strconv.Itoa(resJSON.MachineID) + "\n")
						buffer.WriteString(strconv.FormatBool(resJSON.IsMaster) + "\n")

						file.Write(buffer.Bytes())

						newP, err := cluster.LoadPartikle(c, resJSON.Name)
						if err != nil {
							logger.Fatal("Node.Create", "Node cannot be loaded from new config")
						}
						if err = newP.GenerateDockerCerts(c.DockerConfigPath()); err != nil {
							logger.Fatal("Node.Create", "Node cannot be loaded from new config")
						}

						if err = newP.UploadConsulCerts("/etc/docker/"); err != nil {
							logger.Fatal("Node.Create", "Cannot upload Consul certs : "+err.Error())
						}

						if err = newP.UploadDockerCerts(); err != nil {
							logger.Fatal("Node.Create", "Cannot upload Docker certs : "+err.Error())
						}

						if err = newP.StartDocker(); err != nil {
							logger.Fatal("Node.Create", "Cannot start Docker : "+err.Error())
						}
						if err = newP.WaitDocker(); err != nil {
							logger.Fatal("Node.Create", "Docker cannot be detected : "+err.Error())
						}

						logger.Info("Node.Create", "OK!")
					}

				}
			}
		}
	},
}

func init() {
	nodeCmd.AddCommand(createCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// createCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// createCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
