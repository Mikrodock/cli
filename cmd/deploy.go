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
	"fmt"
	"io/ioutil"
	"mikrodock-cli/cluster"
	"mikrodock-cli/logger"
	"net/http"
	"path/filepath"

	"github.com/spf13/cobra"
)

var composefile string

// deployCmd represents the deploy command
var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Args: cobra.ExactArgs(2), // The name of the mikrodock cluster - the stack name
	Run: func(cmd *cobra.Command, args []string) {
		absfile, _ := filepath.Abs(composefile)
		c, err := cluster.LoadCluster(args[0])
		if err != nil {
			logger.Fatal("Cluster.Load", "Cannot load cluster "+err.Error())
		} else {
			fmt.Printf("%#v\n", c)
		}
		for _, p := range c.Partikles {
			if p.Name() == "konduktor" {
				filecnt, err := ioutil.ReadFile(absfile)
				if err != nil {
					logger.Fatal("Kinetik.Service.Post", err.Error())
				}
				structService := struct {
					StackName            string
					DockerComposeContent string
				}{
					args[1],
					string(filecnt),
				}
				b, err := json.Marshal(structService)
				if err != nil {
					logger.Fatal("Kinetik.Service.Post", err.Error())
				}
				buf := bytes.NewBuffer(b)
				ip := p.IP()
				res, err := http.Post("http://"+ip+":10513/services", "application/json", buf)
				if err != nil {
					logger.Fatal("Kinetik.Service.Post", err.Error())
				}
				if res.StatusCode == 200 {
					logger.Info("Kinetik.Service", "OK!")
				}
			}
		}
	},
}

func init() {
	serviceCmd.AddCommand(deployCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// deployCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	deployCmd.Flags().StringVar(&composefile, "file", "", "The docker compose file")
	deployCmd.MarkFlagFilename("file", "yaml", "yml")
	deployCmd.MarkFlagRequired("file")
}
