package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"sharedlog-stream/pkg/checkpt"
	"strconv"
	"syscall"

	"google.golang.org/grpc"
)

func reusePort(network, address string, conn syscall.RawConn) error {
	return conn.Control(func(fd uintptr) {
		_ = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
	})
}
func checkPort() int {
	var err error
	portStr := os.Getenv("CHKPT_MNGR_PORT")
	var port int
	if portStr == "" {
		port = 6060
	} else {
		port, err = strconv.Atoi(portStr)
		if err != nil {
			log.Fatalf("[FATAL] Failed to read rtx port")
		}
	}
	log.Printf("[INFO] port %v\n", port)
	return port
}

var (
	port = checkPort()
)

func main() {
	config := &net.ListenConfig{Control: reusePort}
	ctx := context.Background()
	lis, err := config.Listen(ctx, "tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("failed to listen %v: %v", port, err)
	}
	s := checkpt.NewCheckpointManagerServer()
	grpcServer := grpc.NewServer()
	checkpt.RegisterChkptMngrServer(grpcServer, s)
	err = grpcServer.Serve(lis)
	if err != nil {
		log.Fatalf("failed to serve grpc server: %v", err)
	}
}
