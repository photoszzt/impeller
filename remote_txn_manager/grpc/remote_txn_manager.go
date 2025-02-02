package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"sharedlog-stream/benchmark/common"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/transaction"
	"sharedlog-stream/pkg/transaction/remote_txn_rpc"
	"strconv"
	"sync/atomic"
	"syscall"
	"time"

	"cs.utexas.edu/zjia/faas"
	"cs.utexas.edu/zjia/faas/types"
	"google.golang.org/grpc"
)

type emptyFuncHandlerFactory struct{}

type emptyFuncHanlder struct {
	env types.Environment
}

var (
	lunched atomic.Bool
	port    int
)

func reusePort(network, address string, conn syscall.RawConn) error {
	return conn.Control(func(fd uintptr) {
		_ = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
	})
}

func (h *emptyFuncHanlder) Call(ctx context.Context, input []byte) ([]byte, error) {
	if !lunched.Load() {
		swapped := lunched.CompareAndSwap(false, true)
		if swapped {
			parsed := &common.RTxnMngrInput{}
			err := json.Unmarshal(input, parsed)
			if err != nil {
				return nil, err
			}
			config := &net.ListenConfig{Control: reusePort}
			lis, err := config.Listen(ctx, "tcp", fmt.Sprintf(":%d", port))
			if err != nil {
				log.Printf("failed to listen %v: %v", port, err)
				return nil, err
			}
			grpcServer := grpc.NewServer()
			remote_txn_rpc.RegisterRemoteTxnMngrServer(grpcServer,
				transaction.NewRemoteTxnManagerServer(h.env, commtypes.SerdeFormat(parsed.SerdeFormat)))
			go func() {
				err = grpcServer.Serve(lis)
				if err != nil {
					log.Fatalf("failed to serve grpc server: %v", err)
				}
			}()
		}
	}
	timeout := time.Duration(300) * time.Second
	start := time.Now()
	for {
		time.Sleep(time.Duration(10) * time.Second)
		if time.Since(start) > timeout {
			break
		}
	}
	return nil, nil
}

func (f *emptyFuncHandlerFactory) New(env types.Environment, funcName string) (types.FuncHandler, error) {
	return &emptyFuncHanlder{
		env: env,
	}, nil
}

func (f *emptyFuncHandlerFactory) GrpcNew(env types.Environment, service string) (types.GrpcFuncHandler, error) {
	return nil, fmt.Errorf("not implemented")
}

func main() {
	var err error
	portStr := os.Getenv("RTX_PORT")
	if portStr == "" {
		port = 5050
	} else {
		port, err = strconv.Atoi(portStr)
		if err != nil {
			log.Fatalf("[FATAL] Failed to read rtx port")
		}
	}
	log.Printf("[INFO] port %v\n", port)
	factory := &emptyFuncHandlerFactory{}
	lunched.Store(false)
	faas.Serve(factory)
}
