package main

import (
	"context"
	"fmt"
	"time"

	"gitee.com/chunanyong/zorm"
	"github.com/seata/seata-go/pkg/client"
	"github.com/seata/seata-go/pkg/tm"
)

func main() {
	client.InitPath("./conf/seatago.yml")
	initService()
	tm.WithGlobalTx(context.Background(), &tm.GtxConfig{
		Name:    "ATSampleLocalGlobalTx",
		Timeout: time.Second * 30,
	}, insertData)

	<-make(chan struct{})
}

func insertData(ctx context.Context) error {
	sql := "INSERT INTO `order_tbl` (`id`, `user_id`, `commodity_code`, `count`, `money`, `descs`) VALUES (?, ?, ?, ?, ?, ?);"
	rows, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		finder := zorm.NewFinder().Append(sql, 333, "NO-100001", "C100000", 100, nil, "init desc")
		return zorm.UpdateFinder(ctx, finder)
	})
	if err != nil {
		fmt.Printf("insert failed, err:%v\n", err)
		return err
	}
	fmt.Printf("insert success： %d.\n", rows)
	return nil
}

func deleteData(ctx context.Context) error {
	sql := "delete from order_tbl where id=?"
	rows, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		finder := zorm.NewFinder().Append(sql, 2)
		return zorm.UpdateFinder(ctx, finder)
	})

	if err != nil {
		fmt.Printf("delete failed, err:%v\n", err)
		return err
	}
	fmt.Printf("delete success： %d.\n", rows)
	return nil
}

func updateData(ctx context.Context) error {
	sql := "update order_tbl set descs=? where id=?"
	rows, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		finder := zorm.NewFinder().Append(sql, fmt.Sprintf("NewDescs-%d", time.Now().UnixMilli()), 1)
		return zorm.UpdateFinder(ctx, finder)
	})
	if err != nil {
		fmt.Printf("update failed, err:%v\n", err)
		return err
	}
	fmt.Printf("update success： %d.\n", rows)
	return nil
}
