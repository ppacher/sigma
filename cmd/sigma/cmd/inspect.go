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
	"time"

	"github.com/homebot/core/urn"
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
		var u urn.URN

		if inspectURN != "" {
			u = urn.URN(inspectURN)
		}

		if inspectName != "" {
			if u != "" {
				log.Fatal("only --name or --urn can be specified")
			}

			u = urn.SigmaFunctionResource.BuildURN("", "", inspectName)
		}

		cli, conn, err := getClient()
		if err != nil {
			log.Fatal(err)
		}
		defer conn.Close()

		ctx, _ := getContext(context.Background())
		res, err := cli.Inspect(ctx, urn.ToProtobuf(u))
		if err != nil {
			log.Fatal(err)
		}

		spec := sigma.SpecFromProto(res.Spec)

		fmt.Printf("Resource-ID: %s\n", u.String())
		fmt.Printf("Type: %s\n", spec.Type)

		if !inspectVerbose {
			fmt.Println("")
			for _, n := range res.Nodes {
				nodeURN := urn.FromProtobuf(n.GetUrn())
				nodeID := nodeURN[len(u.String())+1:]
				fmt.Printf("%s:\t%s\t% 3d invocations\n", nodeID, n.GetState().String(), n.Statistics.Invocations)
			}
		} else {
			for _, n := range res.Nodes {
				fmt.Printf("\n[%s]\n", urn.FromProtobuf(n.GetUrn()).String())
				fmt.Printf("\tState: %s\n", n.State.String())
				fmt.Printf("\tCreated: %s\n", time.Unix(0, n.Statistics.CreatedAt))
				fmt.Printf("\tInvocations: %d\n", n.Statistics.Invocations)
				fmt.Printf("\tLast-Invocation: %s\n", time.Unix(0, n.Statistics.LastInvocation).String())
				fmt.Printf("\tMean-Execution-Time: %s\n", time.Duration(n.Statistics.MeanExecutionTime))
				fmt.Printf("\tTotal-Execution-Time: %s\n", time.Duration(n.Statistics.TotalExecutionTime))
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
