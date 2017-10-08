// Copyright © 2017 NAME HERE <EMAIL ADDRESS>
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
	"context"
	"log"

	sigmaV1 "github.com/homebot/protobuf/pkg/api/sigma/v1"

	"github.com/spf13/cobra"
)

var (
	destroyName string
	destroyURN  string
)

// destroyCmd represents the destroy command
var destroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "Destroy a function deployed on Sigma",
	Run: func(cmd *cobra.Command, args []string) {
		var target string

		if destroyURN != "" && destroyName != "" {
			log.Fatal("only --name or --urn can be specified")
		}

		if destroyURN != "" {
			target = destroyURN
		}

		cli, conn, err := getClient()
		if err != nil {
			log.Fatal(err)
		}
		defer conn.Close()

		ctx, _ := getContext(context.Background())

		_, err = cli.Destroy(ctx, &sigmaV1.DestroyRequest{
			Name: target,
		})
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("Function %s destroyed", target)
	},
}

func init() {
	RootCmd.AddCommand(destroyCmd)

	destroyCmd.Flags().StringVarP(&destroyName, "name", "n", "", "Name of the function to destroy")
	destroyCmd.Flags().StringVarP(&destroyURN, "urn", "u", "", "URN of the function to destroy")
}
