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
	"log"
	"net"
	"os"
	"strings"

	"google.golang.org/grpc"

	"github.com/homebot/idam/policy"
	"github.com/homebot/insight/logger"
	sigmaV1 "github.com/homebot/protobuf/pkg/api/sigma/v1"
	"github.com/homebot/sigma/cmd/sigma/config"
	"github.com/homebot/sigma/launcher"
	"github.com/homebot/sigma/launcher/docker"
	"github.com/homebot/sigma/launcher/process"
	"github.com/homebot/sigma/node"
	"github.com/homebot/sigma/scheduler"
	"github.com/homebot/sigma/server"
	"github.com/spf13/cobra"
)

var (
	serverConfigPath string
	logEvents        bool
)

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start or configure the sigma server",
	Long:  `This command allows to manage the built-in sigma server.`,
	Run: func(cmd *cobra.Command, args []string) {
		if logEvents {
			log.Fatal("--log-events not yet supported")
		}

		f, err := os.Open(serverConfigPath)
		if err != nil {
			log.Fatal(err)
		}

		var c *config.Config

		if strings.HasSuffix(serverConfigPath, "yaml") {
			c, err = config.ReadYAML(f)
		} else if strings.HasSuffix(serverConfigPath, "json") {
			c, err = config.ReadJSON(f)
		} else {
			log.Fatal("unknown configuration file format. Expected JSON or YAML")
		}

		if err != nil {
			log.Fatal(err)
		}

		if err := c.Valid(); err != nil {
			log.Fatal(err)
		}

		launcher := getLauncher(*c)
		if launcher == nil {
			log.Fatal("Invalid or no launcher configured")
		}

		nodeServer := node.NewNodeServer()
		deployer := node.NewDeployer(nodeServer, launcher, c.Nodes.Listen)
		scheduler, err := scheduler.NewScheduler(deployer)
		if err != nil {
			log.Fatal(err)
		}
		server, err := server.NewServer(scheduler)
		if err != nil {
			log.Fatal(err)
		}

		grpcNodeListener, err := net.Listen("tcp", c.Nodes.Listen)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("node handler server running on %s\n", grpcNodeListener.Addr())

		grpcServerListener, err := net.Listen("tcp", c.Server.Listen)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("sigma server running on %s\n", grpcServerListener.Addr())

		grpcNodeServer := grpc.NewServer()
		sigmaV1.RegisterNodeHandlerServer(grpcNodeServer, nodeServer)

		l, err := logger.NewInsightLogger(logger.WithServiceType("sigma"))
		if err != nil {
			log.Fatal(err)
		}

		p, err := policy.NewEnforcer("homebot/api/sigma/v1/sigma.proto")
		if err != nil {
			log.Fatal(err)
		}
		p.SetLogger(l)

		grpcSigmaServer := grpc.NewServer(p.ServerOptions()...)
		sigmaV1.RegisterSigmaServer(grpcSigmaServer, server)

		ch := make(chan struct{})
		go func() {
			defer close(ch)
			if err := grpcNodeServer.Serve(grpcNodeListener); err != nil {
				log.Fatal(err)
			}
		}()

		go func() {
			defer close(ch)
			if err := grpcSigmaServer.Serve(grpcServerListener); err != nil {
				log.Fatal(err)
			}
		}()

		<-ch
	},
}

func init() {
	RootCmd.AddCommand(serverCmd)

	serverCmd.Flags().StringVarP(&serverConfigPath, "cfg", "c", "./sigma.yaml", "Path to Sigma server configuration file")
	serverCmd.Flags().BoolVar(&logEvents, "log-events", false, "Log events to stderr")
}

func getLauncher(c config.Config) launcher.Launcher {
	if c.Launchers.Process != nil {
		types := make(map[string]process.TypeConfig)

		for key, cfg := range c.Launchers.Process.Types {
			types[key] = process.TypeConfig{
				Command: cfg.Command,
			}
		}

		launcher := process.NewLauncher(types)

		return launcher
	}

	if c.Launchers.Docker != nil {
		launcher, err := docker.New(*c.Launchers.Docker)
		if err != nil {
			log.Fatal(err)
		}

		return launcher
	}

	return nil
}
