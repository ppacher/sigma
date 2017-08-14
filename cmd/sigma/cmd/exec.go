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
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/homebot/core/urn"

	sigma_api "github.com/homebot/protobuf/pkg/api/sigma"
	"github.com/homebot/sigma"
	"github.com/spf13/cobra"
)

var (
	execFunctionName string
	execFunctionURN  string
	execEventType    string
	execEventPayload string
)

// execCmd represents the exec command
var execCmd = &cobra.Command{
	Use:   "exec",
	Short: "Execute a function",
	Run: func(cmd *cobra.Command, args []string) {
		target := urn.URN("")
		if execFunctionName == "" && execFunctionURN == "" {
			log.Fatalf("Either --name or --urn must be specified")
		}

		if execFunctionName != "" {
			target = urn.SigmaFunctionResource.BuildURN("", "", execFunctionName)
		}

		if execFunctionURN != "" && target.String() != "" {
			log.Fatalf("Only --name or --urn can be specified")
		}

		if execFunctionURN != "" {
			target = urn.URN(execFunctionURN)
		}

		if execEventType == "" {
			log.Fatal("--type is missing")
		}

		e := sigma.NewSimpleEvent(execEventType, []byte(execEventPayload))

		cli, conn, err := getClient()
		if err != nil {
			log.Fatal(err)
		}
		defer conn.Close()

		res, err := cli.Dispatch(context.Background(), &sigma_api.DispatchRequest{
			Target: urn.ToProtobuf(target),
			Event: &sigma_api.DispatchEvent{
				Id:      e.Type(),
				Payload: e.Payload(),
			},
		})
		if err != nil {
			log.Fatal(err)
		}

		if res.GetError() != "" {
			log.Fatal(errors.New(res.GetError()))
		}

		fmt.Printf("Node: %s\n\n", urn.FromProtobuf(res.GetNode()).String())
		fmt.Println(string(res.GetData()))
	},
}

func init() {
	RootCmd.AddCommand(execCmd)

	execCmd.Flags().StringVarP(&execFunctionName, "name", "n", "", "The name of the function to execute")
	execCmd.Flags().StringVarP(&execFunctionURN, "urn", "u", "", "The URN of the function to execute")
	execCmd.Flags().StringVarP(&execEventType, "type", "t", "", "The event type to publish")
	execCmd.Flags().StringVarP(&execEventPayload, "payload", "d", "", "The data to send to the function")
}
