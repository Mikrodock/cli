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
	"mikrodock-cli/cluster"
	"mikrodock-cli/logger"
	"os"
	"strings"

	"github.com/lextoumbourou/goodhosts"
	"github.com/spf13/cobra"
)

// hostsCmd represents the hosts command
var hostsCmd = &cobra.Command{
	Use:   "hosts",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		uid := os.Geteuid()
		if uid != 0 {
			logger.Fatal("Hosts", "Sorry but your must run this command as root")
		}

		hosts, err := goodhosts.NewHosts()
		if err != nil {
			logger.Fatal("Hosts", err.Error())
		}
		c, err := cluster.LoadCluster(args[0])
		if err != nil {
			logger.Fatal("Cluster.Load", err.Error())
		}

		switch args[1] {
		case "add":
			DeleteEntries(c.Name, hosts)
			AddEntries(c, hosts)
			break
		case "rm":
			DeleteEntries(c.Name, hosts)
			break
		}
	},
}

func DeleteEntries(clusterName string, h goodhosts.Hosts) error {
	for _, l := range h.Lines {
		deleteThisLine := false
		for _, d := range l.Hosts {
			if strings.HasSuffix(d, "."+clusterName) {
				logger.Info("Hosts", "Delete line "+l.Raw)
				deleteThisLine = true
			}
		}
		if deleteThisLine {
			h.Remove(l.IP, l.Hosts...)
			h.Flush()
		}
	}
	return nil
}

func AddEntries(cluster *cluster.Cluster, h goodhosts.Hosts) error {
	for _, p := range cluster.Partikles {
		logger.Info("Hosts", "Add line "+p.Name()+"."+cluster.Name)
		err := h.Add(p.IP(), p.Name()+"."+cluster.Name)
		if err != nil {
			logger.Warn("Hosts", err.Error())
		}
	}
	err := h.Flush()
	if err != nil {
		logger.Warn("Hosts", err.Error())
	}
	return nil
}

func init() {
	rootCmd.AddCommand(hostsCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// hostsCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// hostsCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
