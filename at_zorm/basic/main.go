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
	client.Init()
	initService()
	tm.WithGlobalTx(context.Background(), &tm.GtxConfig{
		Name:    "ATSampleLocalGlobalTx",
		Timeout: time.Second * 30,
	}, updateData)

	<-make(chan struct{})
}

func updateData(ctx context.Context) error {
	fmt.Printf("===================== xid=%s =====================\n", tm.GetXID(ctx))
	sql := "update order_tbl set descs=? where id=?"
	rows, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		finder := zorm.NewFinder().Append(sql, fmt.Sprintf("NewDescs1-%d", time.Now().UnixMilli()), 1)
		return zorm.UpdateFinder(ctx, finder)
	})
	if err != nil {
		fmt.Printf("update failed, err:%v\n", err)
		return err
	}
	fmt.Printf("update success： %d.\n", rows)
	return nil
}
