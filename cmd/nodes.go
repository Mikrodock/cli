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
	"sort"
	"strconv"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

// nodesCmd represents the nodes command
var nodesCmd = &cobra.Command{
	Use:   "nodes",
	Short: "Show nodes of the cluster",
	Long:  ``,
	Args:  cobra.ExactArgs(1), // Cluster name
	Run: func(cmd *cobra.Command, args []string) {
		c, err := cluster.LoadCluster(args[0])
		if err != nil {
			logger.Fatal("Cluster.Load", "Cannot load cluster test")
		}
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Name", "IP", "Master"})
		table.AppendBulk(clusterTable(c))
		table.Render()
	},
}

func init() {
	showCmd.AddCommand(nodesCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// nodesCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// nodesCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func clusterTable(c *cluster.Cluster) [][]string {
	tContent := make([][]string, len(c.Partikles))
	sort.Slice(c.Partikles, func(i, j int) bool {
		return c.Partikles[i].Name() < c.Partikles[j].Name()
	})
	for i, p := range c.Partikles {
		tLine := make([]string, 3)
		tLine[0] = p.Name()
		tLine[1] = p.IP()
		tLine[2] = strconv.FormatBool(p.IsMaster)
		tContent[i] = tLine
	}
	return tContent
}
