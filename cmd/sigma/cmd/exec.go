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

	sigmaV1 "github.com/homebot/protobuf/pkg/api/sigma/v1"
	"github.com/homebot/sigma"
	"github.com/spf13/cobra"
)

var (
	execFunctionName string
	execFunctionURN  string
	execEventType    string
	execEventPayload string
	execVerbose      bool
)

// execCmd represents the exec command
var execCmd = &cobra.Command{
	Use:   "exec",
	Short: "Execute a function",
	Run: func(cmd *cobra.Command, args []string) {
		target := ""
		if execFunctionName == "" && execFunctionURN == "" {
			log.Fatalf("Either --name or --urn must be specified")
		}

		if execFunctionURN != "" && target != "" {
			log.Fatalf("Only --name or --urn can be specified")
		}

		if execFunctionURN != "" {
			target = execFunctionURN
		}

		e := sigma.NewSimpleEvent(execEventType, []byte(execEventPayload))

		cli, conn, err := getClient()
		if err != nil {
			log.Fatal(err)
		}
		defer conn.Close()

		ctx, _ := getContext(context.Background())

		res, err := cli.Dispatch(ctx, &sigmaV1.DispatchRequest{
			Target: target,
			Event: &sigmaV1.DispatchEvent{
				Type:    e.Type(),
				Payload: e.Payload(),
			},
		})
		if err != nil {
			log.Fatal(err)
		}

		if res.GetError() != "" {
			log.Fatal(errors.New(res.GetError()))
		}

		if execVerbose {
			fmt.Printf("Node: %s\n\n", res.GetNode())
		}
		fmt.Println(string(res.GetData()))
	},
}

func init() {
	RootCmd.AddCommand(execCmd)

	execCmd.Flags().StringVarP(&execFunctionName, "name", "n", "", "The name of the function to execute")
	execCmd.Flags().StringVarP(&execFunctionURN, "urn", "u", "", "The URN of the function to execute")
	execCmd.Flags().StringVarP(&execEventType, "type", "t", "", "The event type to publish")
	execCmd.Flags().StringVarP(&execEventPayload, "payload", "d", "", "The data to send to the function")
	execCmd.Flags().BoolVarP(&execVerbose, "verbose", "v", false, "Disable versbose output")
}
