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
	"encoding/json"
	"fmt"
	"log"

	"github.com/homebot/protobuf/pkg/api"
	"github.com/spf13/cobra"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List registered functions",
	Run: func(cmd *cobra.Command, args []string) {
		cli, conn, err := getClient()
		if err != nil {
			log.Fatal(err)
		}
		defer conn.Close()

		res, err := cli.List(context.Background(), &api.Empty{})
		if err != nil {
			log.Fatal(err)
		}

		blob, _ := json.MarshalIndent(res, "", "  ")

		fmt.Println(string(blob))
	},
}

func init() {
	RootCmd.AddCommand(listCmd)
}
