// Copyright © 2017 The IoT-Cloud Authors
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
	"io/ioutil"
	"log"
	"os"

	yaml "github.com/ghodss/yaml"
	"github.com/homebot/sigma/cmd/sigma/config"
	"github.com/homebot/sigma/cmd/sigma/scaffolding"
	"github.com/spf13/cobra"
)

var (
	launcherType string
	nodeTypes    []string
)

// serverInitCmd represents the init command
var serverInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new Sigma server configuration file",
	Run: func(cmd *cobra.Command, args []string) {
		c := config.Config{
			Server: config.SigmaServerConfig{
				Listen: "localhost:50051",
			},
			Nodes: config.NodeServerConfig{
				Listen:           "localhost:50052",
				AdvertiseAddress: "127.0.0.1:50052",
			},
		}
		path := "./sigma.yaml"

		if content, err := ioutil.ReadFile(path); err == nil {
			if err := yaml.Unmarshal(content, &c); err != nil {
				log.Fatal(err)
			}
		}

		if len(nodeTypes) > 0 && launcherType == "" {
			log.Fatal("Missing --launcher parameter (mandatory for --add-type)")
		}

		if launcherType != "" {
			if err := scaffolding.CreateLauncher(launcherType, &c, nodeTypes); err != nil {
				log.Fatal(err)
			}
		}

		f, err := os.Create(path)
		if err != nil {
			log.Fatal(err)
		}

		if err := c.WriteYAML(f); err != nil {
			log.Fatal(err)
		}

		log.Println("Sigma server configuration created at './sigma.yaml'")
	},
}

func init() {
	serverCmd.AddCommand(serverInitCmd)

	serverInitCmd.Flags().StringVarP(&launcherType, "launcher", "l", "", "The launcher to configure for the new server")
	serverInitCmd.Flags().StringSliceVarP(&nodeTypes, "add-type", "a", nil, "A list of node execution types to configure")
}
