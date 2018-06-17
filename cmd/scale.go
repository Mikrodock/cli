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
	"fmt"
	"io/ioutil"
	"mikrodock-cli/cluster"
	"mikrodock-cli/logger"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/spf13/cobra"
)

// scaleCmd represents the scale command
var scaleCmd = &cobra.Command{
	Use:   "scale",
	Short: "Scale up or down a service of a stack",
	Long:  ``,
	Args:  cobra.ExactArgs(5), // The name of the mikrodock cluster - the stack name - the service - UP OR DOWN - qty
	Run: func(cmd *cobra.Command, args []string) {
		c, err := cluster.LoadCluster(args[0])
		if err != nil {
			logger.Fatal("Cluster.Load", "Cannot load cluster "+err.Error())
		} else {
			fmt.Printf("%#v\n", c)
		}

		qty, _ := strconv.Atoi(args[4])

		var wg sync.WaitGroup
		wg.Add(qty)

		for _, p := range c.Partikles {
			if p.Name() == "konduktor" {
				ip := p.IP()
				logger.Info("Kinetik.Service.Scale", "http://"+ip+":10513/services/"+args[1]+"/"+args[2]+"/scale/"+args[3])

				for i := 0; i < qty; i++ {
					go func(waitG *sync.WaitGroup) {
						defer waitG.Done()
						res, err := http.Post("http://"+ip+":10513/services/"+args[1]+"/"+args[2]+"/scale/"+args[3], "application/json", bytes.NewBuffer([]byte{}))
						if err != nil {
							logger.Fatal("Kinetik.Service.Post", err.Error())
						}
						body, _ := ioutil.ReadAll(res.Body)
						if res.StatusCode == 200 {
							logger.Info("Kinetik.Service", "OK! "+string(body))
						} else {
							logger.Fatal("Kinetik.Service", string(body))
						}
					}(&wg)

					time.Sleep(100 * time.Millisecond)
				}

				wg.Wait()
			}
		}

		wg.Wait()
	},
}

func init() {
	serviceCmd.AddCommand(scaleCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// scaleCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// scaleCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
