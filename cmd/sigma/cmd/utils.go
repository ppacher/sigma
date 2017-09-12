package cmd

import (
	"context"

	"github.com/homebot/idam/token"
	"github.com/homebot/protobuf/pkg/api/sigma"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func getClient() (sigma.SigmaClient, *grpc.ClientConn, error) {
	addr := sigmaServerAddress
	if addr == "" {
		addr = "localhost:50051"
	}
	conn, err := grpc.Dial(addr, grpc.WithInsecure())

	if err != nil {
		return nil, nil, err
	}

	return sigma.NewSigmaClient(conn), conn, nil
}

func getContext(ctx context.Context) (context.Context, string) {
	// try to read the IDAM token file
	var paths []string

	if idamTokenFile != "" {
		paths = append(paths, idamTokenFile)
	}

	t, path, err := token.LoadToken(paths)
	if err != nil {
		return ctx, ""
	}

	md := metadata.New(map[string]string{
		"authorization": t,
	})

	return metadata.NewOutgoingContext(ctx, md), path
}
