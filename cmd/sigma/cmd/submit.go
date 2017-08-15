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
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	yaml "github.com/ghodss/yaml"
	"github.com/homebot/core/urn"
	sigma_api "github.com/homebot/protobuf/pkg/api/sigma"
	"github.com/homebot/sigma"

	"github.com/spf13/cobra"
)

// FunctionSpec represents the JSON/YAML file format for
// sigma function specifications
type FunctionSpec struct {
	sigma.FunctionSpec

	// Content describes where to fetch the functions content from
	Content struct {
		// Inline specifies the function content directly within the spec file
		Inline string `json:"inline" yaml:"inline"`

		// File is the path to the function handler file to transmit during "submit"
		File string `json:"file" yaml:"file"`
	}
}

// submitCmd represents the submit command
var submitCmd = &cobra.Command{
	Use:   "submit",
	Short: "Sumbit a function to the Sigma server",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			log.Fatal(errors.New("expected one argument: function-name"))
		}

		funcName := args[0]

		path := fmt.Sprintf("%s.yaml", funcName)
		f, err := os.Open(path)
		if err != nil {
			log.Fatal(err)
		}

		content, err := ioutil.ReadAll(f)
		if err != nil {
			log.Fatal(err)
		}

		var spec FunctionSpec
		if err := yaml.Unmarshal(content, &spec); err != nil {
			log.Fatal(err)
		}

		if spec.Content.Inline != "" && spec.Content.File != "" {
			log.Fatalf("function spec`content`: only `inline` or `file` can be set")
		}

		if spec.Content.File != "" {
			data, err := ioutil.ReadFile(spec.Content.File)
			if err != nil {
				log.Fatal(err)
			}

			spec.FunctionSpec.Content = string(data)
		}

		if spec.FunctionSpec.Content == "" {
			log.Fatal("function does not have any content")
		}

		if blob, err := json.MarshalIndent(spec, "", "  "); err == nil {
			fmt.Printf("%s\n", string(blob))
		}

		if blob, err := json.MarshalIndent(spec.FunctionSpec.ToProtobuf(), "", "  "); err == nil {
			fmt.Printf("%s\n", string(blob))
		}

		cli, conn, err := getClient()
		if err != nil {
			log.Fatal(err)
		}
		defer conn.Close()

		res, err := cli.Create(context.Background(), &sigma_api.CreateFunctionRequest{
			Spec: spec.FunctionSpec.ToProtobuf(),
		})

		if err != nil {
			log.Fatal(err)
		}

		u := urn.FromProtobuf(res.GetUrn())
		fmt.Printf("Function created successfully\nURN: %s\n", u.String())
	},
}

func init() {
	RootCmd.AddCommand(submitCmd)
}
