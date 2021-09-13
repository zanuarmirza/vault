package grpc

import (
	"zanuarmirza/vault/pb"
	vault "zanuarmirza/vault/service"

	grpctransport "github.com/go-kit/kit/transport/grpc"
	"google.golang.org/grpc"
)

// New makes a new vault.Service client.
func New(conn *grpc.ClientConn) vault.Service {
	var hashEndpoint = grpctransport.NewClient(
		conn, "pb.Vault", "Hash",
		vault.EncodeGRPCHashRequest,
		vault.DecodeGRPCHashResponse,
		pb.HashResponse{},
	).Endpoint()
	var validateEndpoint = grpctransport.NewClient(
		conn, "pb.Vault", "Validate",
		vault.EncodeGRPCValidateRequest,
		vault.DecodeGRPCValidateResponse,
		pb.ValidateResponse{},
	).Endpoint()
	return vault.Endpoints{
		HashEndpoint:     hashEndpoint,
		ValidateEndpoint: validateEndpoint,
	}
}
