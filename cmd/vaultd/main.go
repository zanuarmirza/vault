package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"zanuarmirza/vault/pb"
	vault "zanuarmirza/vault/service"

	"github.com/go-kit/kit/endpoint"
	"github.com/juju/ratelimit"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var ErrLimitExceed = errors.New("Rate Limit Exceed")

func NewTokenBucketLimiter(tb *ratelimit.Bucket) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			if tb.TakeAvailable(1) == 0 {
				return nil, ErrLimitExceed
			}
			return next(ctx, request)
		}
	}
}

// grpc server
func main() {
	var (
		httpAddr = flag.String("http", ":8080", "http listen address")
		gRPCAddr = flag.String("grpc", ":8081", "gRPC listen address")
	)
	flag.Parse()
	ctx := context.Background()
	srv := vault.NewService()
	errChan := make(chan error)
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errChan <- fmt.Errorf("%s", <-c)
	}()

	rlbucket := ratelimit.NewBucket(1*time.Second, 5)
	hashEndpoint := vault.MakeHashEndpoint(srv)
	{
		hashEndpoint = NewTokenBucketLimiter(rlbucket)(hashEndpoint)
	}
	validateEndpoint := vault.MakeValidateEndpoint(srv)
	{
		validateEndpoint = NewTokenBucketLimiter(rlbucket)(validateEndpoint)
	}
	endpoints := vault.Endpoints{
		HashEndpoint:     hashEndpoint,
		ValidateEndpoint: validateEndpoint,
	}

	// HTTP transport
	go func() {
		log.Println("http:", *httpAddr)
		handler := vault.NewHTTPServer(endpoints)
		errChan <- http.ListenAndServe(*httpAddr, handler)
	}()

	// gRPC transport
	go func() {
		listener, err := net.Listen("tcp", *gRPCAddr)
		if err != nil {
			errChan <- err
			return
		}
		log.Println("grpc:", *gRPCAddr)
		handler := vault.NewGRPCServer(ctx, endpoints)
		gRPCServer := grpc.NewServer()
		pb.RegisterVaultServer(gRPCServer, handler)
		errChan <- gRPCServer.Serve(listener)
	}()

	log.Fatalln(<-errChan)
}
