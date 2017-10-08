package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"

	"google.golang.org/grpc/metadata"

	"google.golang.org/grpc"

	"github.com/homebot/core/utils"
	sigmaV1 "github.com/homebot/protobuf/pkg/api/sigma/v1"
	"github.com/homebot/sigma/launcher"
)

var binary = flag.String("binary", "", "The binary to execute")

type InitMessage struct {
	URN        string         `json:"urn" yaml:"urn"`
	Parameters utils.ValueMap `json:"parameters" yaml:"parameters"`
	Content    []byte         `json:"content" yaml:"content"`
}

func main() {
	flag.Parse()

	if *binary == "" {
		os.Stderr.Write([]byte(fmt.Sprintf("missing -binary argument")))
		return
	}

	c := launcher.ConfigFromEnv()

	f, err := os.Create("/tmp/env")
	if err != nil {
		os.Stderr.Write([]byte(fmt.Sprintf("invalid URN received")))
		return
	}
	f.Write([]byte(fmt.Sprintf("%#v", c)))
	f.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// start binary
	cmd := exec.CommandContext(ctx, *binary)
	for key, value := range c.EnvVars() {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		os.Stderr.Write([]byte(err.Error()))
		return
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		os.Stderr.Write([]byte(err.Error()))
		return
	}
	go io.Copy(os.Stdout, stdout)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		os.Stderr.Write([]byte(err.Error()))
		return
	}
	go io.Copy(os.Stderr, stderr)

	conn, err := grpc.Dial(c.Address, grpc.WithInsecure())
	if err != nil {
		os.Stderr.Write([]byte(err.Error()))
		return
	}
	defer conn.Close()

	cli := sigmaV1.NewNodeHandlerClient(conn)

	md := metadata.Pairs("node-urn", c.URN, "node-secret", c.Secret)
	callCtx := metadata.NewOutgoingContext(ctx, md)

	res, err := cli.Register(callCtx, &sigmaV1.NodeRegistrationRequest{
		Urn:      c.URN,
		NodeType: "dummy",
	})
	if err != nil {
		os.Stderr.Write([]byte(err.Error()))
		return
	}

	if err := cmd.Start(); err != nil {
		os.Stderr.Write([]byte(err.Error()))
		return
	}

	init := InitMessage{
		URN:        res.GetUrn(),
		Content:    res.GetContent(),
		Parameters: utils.ValueMapFrom(res.GetParameters()),
	}

	blob, err := json.Marshal(init)
	if err != nil {
		os.Stderr.Write([]byte(err.Error()))
		return
	}

	stdin.Write(blob)

	stream, err := cli.Subscribe(callCtx)
	if err != nil {
		os.Stderr.Write([]byte(err.Error()))
		return
	}

	go func() {
		defer cancel()
		for {
			msg, err := stream.Recv()
			if err != nil {
				return
			}

			if err := stream.Send(&sigmaV1.ExecutionResult{
				Id: msg.GetId(),
				ExecutionResult: &sigmaV1.ExecutionResult_Result{
					Result: msg.GetPayload(),
				},
			}); err != nil {
				return
			}
		}
	}()

	if err := cmd.Wait(); err != nil {
		os.Stderr.Write([]byte(err.Error()))
		return
	}
}
