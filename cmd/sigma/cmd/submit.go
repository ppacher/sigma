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
	"io/ioutil"
	"log"
	"os"
	"path"
	"strconv"
	"strings"

	yaml "github.com/ghodss/yaml"
	"github.com/homebot/core/urn"
	"github.com/homebot/core/utils"
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
	} `json:"content" yaml:"content"`
}

var (
	intParams    []string
	stringParams []string
	boolParams   []string
	idOverride   string
)

// submitCmd represents the submit command
var submitCmd = &cobra.Command{
	Use:   "submit",
	Short: "Sumbit a function to the Sigma server",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			log.Fatal(errors.New("expected one argument: function-name"))
		}

		funcName := args[0]

		base := ""
		stat, err := os.Stat(funcName)
		if err == nil && stat.IsDir() {
			base = funcName
			funcName = path.Join(base, path.Base(funcName)+".yaml")
		} else if err == nil && !stat.IsDir() {
			base = path.Dir(funcName)
		}

		f, err := os.Open(funcName)
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

		if idOverride != "" {
			spec.ID = idOverride
		}

		if err := parseParameters(spec.Parameteres); err != nil {
			log.Fatal(err)
		}

		if spec.Content.Inline != "" && spec.Content.File != "" {
			log.Fatalf("function spec`content`: only `inline` or `file` can be set")
		}

		if spec.Content.File != "" {
			data, err := ioutil.ReadFile(path.Join(base, spec.Content.File))
			if err != nil {
				log.Fatal(err)
			}

			spec.FunctionSpec.Content = string(data)
		}

		if spec.FunctionSpec.Content == "" {
			log.Fatal("function does not have any content")
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

	submitCmd.Flags().StringSliceVarP(&intParams, "param-int", "i", nil, "Additional parameters in format key=value")
	submitCmd.Flags().StringSliceVarP(&stringParams, "param-str", "s", nil, "Additional parameters in format key=value")
	submitCmd.Flags().StringSliceVarP(&boolParams, "param-bool", "b", nil, "Additional parameters in format key=value")
	submitCmd.Flags().StringVarP(&idOverride, "name", "n", "", "Name for the function to submit. Overrides values from the spec")
}

func parseParameters(m utils.ValueMap) error {
	for _, v := range intParams {
		k, i, err := splitInt(v)
		if err != nil {
			return err
		}

		m[k] = i
	}

	for _, v := range boolParams {
		k, b, err := splitBool(v)
		if err != nil {
			return err
		}

		m[k] = b
	}

	for _, v := range stringParams {
		k, s, err := splitString(v)
		if err != nil {
			return nil
		}

		m[k] = s
	}

	return nil
}

func splitInt(k string) (string, int, error) {
	parts := strings.Split(k, "=")
	if len(parts) < 2 {
		return "", 0, errors.New("invalid parameter")
	}

	key := parts[0]
	value, err := strconv.ParseInt(parts[1], 10, 64)
	return key, int(value), err
}

func splitBool(k string) (string, bool, error) {
	parts := strings.Split(k, "=")
	if len(parts) < 2 {
		return "", false, errors.New("invalid parameter")
	}

	key := parts[0]
	switch parts[1] {
	case "true", "t", "1", "on":
		return key, true, nil
	case "false", "f", "0", "off":
		return key, false, nil
	}

	return key, false, errors.New("invalid parameter")
}

func splitString(k string) (string, string, error) {
	parts := strings.Split(k, "=")
	if len(parts) < 2 {
		return "", "", errors.New("invalid parameter")
	}

	key := parts[0]
	return key, strings.Join(parts[1:], "="), nil
}
