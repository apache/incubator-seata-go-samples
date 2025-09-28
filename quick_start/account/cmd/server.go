package main

import (
	"fmt"
	"google.golang.org/grpc"
	"net"
	"seata.apache.org/seata-go-samples/quick_start/account/server"
	pb "seata.apache.org/seata-go-samples/quick_start/api"
	grpc2 "seata.apache.org/seata-go/pkg/integration/grpc"
	"seata.apache.org/seata-go/pkg/util/log"
)

func main() {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", 50051))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer(grpc.UnaryInterceptor(grpc2.ServerTransactionInterceptor))
	pb.RegisterAccountServiceServer(s, &server.AccountServer{})
	log.Infof("business listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
