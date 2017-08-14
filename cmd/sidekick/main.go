package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"

	"google.golang.org/grpc/metadata"

	"google.golang.org/grpc"

	"github.com/homebot/core/urn"
	sigma_api "github.com/homebot/protobuf/pkg/api/sigma"
	"github.com/homebot/sigma"
	"github.com/homebot/sigma/launcher"
)

var binary = flag.String("binary", "", "The binary to execute")

// Registration is sent on the targets stdin as soon as the registration
// is successfull
type Registration struct {
	Spec sigma.FunctionSpec `json:"spec"`
}

func main() {
	flag.Parse()

	if *binary == "" {
		os.Stderr.Write([]byte(fmt.Sprintf("missing -binary argument")))
		return
	}

	c := launcher.ConfigFromEnv()

	if !c.URN.Valid() {
		os.Stderr.Write([]byte(fmt.Sprintf("invalid URN received: %s", c.URN)))
		return
	}

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

	cli := sigma_api.NewNodeHandlerClient(conn)

	md := metadata.Pairs("node-urn", c.URN.String(), "node-secret", c.Secret)
	callCtx := metadata.NewOutgoingContext(ctx, md)

	res, err := cli.Register(callCtx, &sigma_api.NodeRegistrationRequest{
		Urn:      urn.ToProtobuf(c.URN),
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

	stdin.Write([]byte(urn.FromProtobuf(res.GetUrn()).String()))

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

			if err := stream.Send(&sigma_api.ExecutionResult{
				Id: msg.GetId(),
				ExecutionResult: &sigma_api.ExecutionResult_Result{
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
