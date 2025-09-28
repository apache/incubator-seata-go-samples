package server

import (
	"context"
	"fmt"
	"strconv"

	"seata.apache.org/seata-go-samples/quick_start/account/model"
	"seata.apache.org/seata-go-samples/quick_start/account/service"
	pb "seata.apache.org/seata-go-samples/quick_start/api"
)

type AccountServer struct {
	svc *service.AccountService
	pb.UnimplementedAccountServiceServer
}

func (a *AccountServer) Deduct(ctx context.Context, request *pb.AccountDeductRequest) (*pb.AccountResponse, error) {
	userID, err := strconv.ParseInt(request.UserId, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid user_id: %w", err)
	}

	account := model.Account{
		UserID:  userID,
		Balance: request.Money,
	}

	if err := a.svc.Deduct(ctx, account); err != nil {
		return nil, err
	}

	return &pb.AccountResponse{
		UserId:       request.UserId,
		Balance:      0,
		FreezeAmount: 0,
	}, nil
}
