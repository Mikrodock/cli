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
	"fmt"
	"mikrodock-cli/cluster"
	"mikrodock-cli/logger"
	"os"
	"sync"

	"github.com/spf13/cobra"
)

// destroyCmd represents the destroy command
var destroyCmd = &cobra.Command{
	Use:   "destroy",
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
			logger.Fatal("Cluster.Load", "Cannot load cluster "+err.Error())
		} else {
			var wg sync.WaitGroup
			wg.Add(len(c.Partikles))
			logger.Debug("Cluster.Partikles", fmt.Sprintf("%#v", c.Partikles))
			for _, part := range c.Partikles {
				go func(partikle *cluster.Partikle, wg *sync.WaitGroup) {
					logger.Debug("Cluster.Partikle.Destroy", fmt.Sprintf("%#v", partikle))
					err := partikle.Driver.Destroy()
					if err != nil {
						logger.Info("Cluster.Partikle.Destroy", "Cannot destroy droplet : "+err.Error())
					}
					wg.Done()
				}(part, &wg)
			}
			err := os.RemoveAll(c.DeployDir)
			if err != nil {
				logger.Fatal("Cluster.Destroy", "Error while deleting cluster directory : "+err.Error())
			}

			wg.Wait()

			logger.Info("Cluster.Destroy", "Cluster destroyed")
		}
	},
}

func init() {
	rootCmd.AddCommand(destroyCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// destroyCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// destroyCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
