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
	"mikrodock-cli/cluster"
	"mikrodock-cli/logger"
	"net/http"
	"os"
	"sort"
	"strconv"

	kModels "github.com/mikrodock/kinetik-server/models"

	"github.com/docker/cli/cli/compose/types"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

// servicesCmd represents the services command
var servicesCmd = &cobra.Command{
	Use:   "services",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		services := make([]kModels.Service, 0)

		c, _ := cluster.LoadCluster(args[0])
		for _, p := range c.Partikles {
			if p.Name() == "konduktor" {
				fmt.Println("http://" + p.IP() + ":10513/services")
				res, err := http.Get("http://" + p.IP() + ":10513/services")
				if err != nil {
					logger.Fatal("Services.Get", err.Error())
				}
				err = json.NewDecoder(res.Body).Decode(&services)
				if err != nil {
					logger.Fatal("Services.Get.Decode", err.Error())
				}
				if len(args) == 1 {
					// Overview mode
					table := tablewriter.NewWriter(os.Stdout)
					table.SetHeader([]string{"Stack name", "Service Name", "Count", "Ports"})
					table.AppendBulk(convertTable(services))
					table.Render()
				} else if len(args) == 2 {
					// Overview mode, filter on stackname
					table := tablewriter.NewWriter(os.Stdout)
					table.SetHeader([]string{"Stack name", "Service Name", "Count", "Ports"})
					deleted := 0
					for i := range services {
						j := i - deleted
						if services[j].StackName != args[1] {
							services = services[:j+copy(services[j:], services[j+1:])]
							deleted++
						}
					}
					table.AppendBulk(convertTable(services))
					table.Render()
				} else {
					// Detail mode, stackname and service name
					var service *kModels.Service
					for _, srv := range services {
						if srv.StackName == args[1] && srv.ServiceName == args[2] {
							service = &srv
							break
						}
					}

					if service != nil {
						table := tablewriter.NewWriter(os.Stdout)
						table.SetHeader([]string{"Host IP", "Instance name"})
						table.AppendBulk(convertDetailTable(service))
						table.Render()
					}
				}
			}
		}
	},
}

func init() {
	showCmd.AddCommand(servicesCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// servicesCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// servicesCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func convertTable(services []kModels.Service) [][]string {
	srvs := make([][]string, len(services))
	for i, srv := range services {
		srvL := make([]string, 4)
		srvL[0] = srv.StackName
		srvL[1] = srv.ServiceName
		srvL[2] = strconv.Itoa(len(srv.Instances))
		srvL[3] = convertPorts(srv.Ports)
		srvs[i] = srvL
	}
	return srvs
}

func convertPorts(ports []types.ServicePortConfig) string {
	buf := bytes.Buffer{}
	for _, port := range ports {
		buf.WriteString(strconv.Itoa(int(port.Published)) + ":" + strconv.Itoa(int(port.Target)) + ";")
	}
	return buf.String()
}

func convertDetailTable(srv *kModels.Service) [][]string {
	a := make([][]string, len(srv.Instances))
	for i, inst := range srv.Instances {
		b := make([]string, 2)
		b[0] = inst.NodeID
		b[1] = inst.ContainerID
		a[i] = b
	}
	sort.Slice(a, func(i, j int) bool {
		return a[i][0] < a[j][0]
	})
	return a
}
