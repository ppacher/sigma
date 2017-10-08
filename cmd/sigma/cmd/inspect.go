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
	"fmt"
	"log"

	"github.com/golang/protobuf/ptypes"

	sigmaV1 "github.com/homebot/protobuf/pkg/api/sigma/v1"
	"github.com/homebot/sigma"
	"github.com/spf13/cobra"
)

var (
	inspectName    string
	inspectURN     string
	inspectVerbose bool
)

// inspectCmd represents the inspect command
var inspectCmd = &cobra.Command{
	Use:   "inspect",
	Short: "Inspect a function running at the sigma server",
	Run: func(cmd *cobra.Command, args []string) {
		var u string

		if inspectURN != "" {
			u = inspectURN
		}

		cli, conn, err := getClient()
		if err != nil {
			log.Fatal(err)
		}
		defer conn.Close()

		ctx, _ := getContext(context.Background())
		res, err := cli.Inspect(ctx, &sigmaV1.InspectRequest{
			Name: u,
		})
		if err != nil {
			log.Fatal(err)
		}

		spec := sigma.SpecFromProto(res.Spec)

		fmt.Printf("Resource-ID: %s\n", u)
		fmt.Printf("Type: %s\n", spec.Type)

		if !inspectVerbose {
			fmt.Println("")
			for _, n := range res.Nodes {
				nodeURN := n.GetUrn()
				nodeID := nodeURN[len(u)+1:]
				fmt.Printf("%s:\t%s\t% 3d invocations\n", nodeID, n.GetState().String(), n.Statistics.Invocations)
			}
		} else {
			for _, n := range res.Nodes {
				created, createdErr := ptypes.Timestamp(n.Statistics.GetCreatedTime())
				last, lastErr := ptypes.Timestamp(n.Statistics.GetLastInvocation())
				mean, meanErr := ptypes.Duration(n.Statistics.GetMeanExecTime())
				total, totalErr := ptypes.Duration(n.Statistics.GetTotalExecTime())

				fmt.Printf("\n[%s]\n", n.GetUrn())
				fmt.Printf("\tState: %s\n", n.State.String())
				if createdErr == nil {
					fmt.Printf("\tCreated: %s\n", created)
				}
				fmt.Printf("\tInvocations: %d\n", n.Statistics.Invocations)
				if lastErr == nil {
					fmt.Printf("\tLast-Invocation: %s\n", last)
				}
				if meanErr == nil {
					fmt.Printf("\tMean-Execution-Time: %s\n", mean)
				}
				if totalErr == nil {
					fmt.Printf("\tTotal-Execution-Time: %s\n", total)
				}
			}
		}
	},
}

func init() {
	RootCmd.AddCommand(inspectCmd)

	inspectCmd.Flags().StringVarP(&inspectName, "name", "n", "", "The name of the function to inspect")
	inspectCmd.Flags().StringVarP(&inspectURN, "urn", "u", "", "The URN of the function to inspect")
	inspectCmd.Flags().BoolVarP(&inspectVerbose, "verbose", "v", false, "Enable verbose output")
}
