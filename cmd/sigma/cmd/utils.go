package cmd

import (
	"github.com/homebot/protobuf/pkg/api/sigma"
	"google.golang.org/grpc"
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
