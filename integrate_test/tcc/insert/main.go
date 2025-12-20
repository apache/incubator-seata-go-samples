/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"context"
	"database/sql"
	"errors"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"seata.apache.org/seata-go/pkg/client"
	"seata.apache.org/seata-go/pkg/rm/tcc"
	"seata.apache.org/seata-go/pkg/tm"
)

type OrderTblModel struct {
	Id            int64  `gorm:"column:id;primaryKey"`
	UserId        string `gorm:"column:user_id"`
	CommodityCode string `gorm:"column:commodity_code"`
	Count         int64  `gorm:"column:count"`
	Money         int64  `gorm:"column:money"`
	Descs         string `gorm:"column:descs"`
}

var gormDB *gorm.DB

type OrderTCCService struct{}

func (o *OrderTCCService) GetActionName() string {
	return "OrderTCCService"
}

func (o *OrderTCCService) Prepare(ctx context.Context, params interface{}) (bool, error) {
	order := params.(OrderTblModel)
	if err := gormDB.WithContext(ctx).Table("order_tbl").Create(&order).Error; err != nil {
		return false, err
	}
	log.Printf("[Prepare] insert order %+v", order)
	return true, nil
}

func (o *OrderTCCService) Commit(ctx context.Context, bac *tm.BusinessActionContext) (bool, error) {
	log.Printf("[Commit] confirm order")
	return true, nil
}

func (o *OrderTCCService) Rollback(ctx context.Context, bac *tm.BusinessActionContext) (bool, error) {
	if err := gormDB.WithContext(ctx).
		Table("order_tbl").
		Where("user_id = ? AND commodity_code = ?", "U10001", "C10001").
		Delete(nil).Error; err != nil {
		return false, err
	}
	log.Printf("[Rollback] cancel order")
	return true, nil
}

func initDB() {
	sqlDB, err := sql.Open("mysql", "root:12345678@tcp(127.0.0.1:3306)/seata_client?parseTime=true")
	if err != nil {
		panic(err)
	}
	gormDB, err = gorm.Open(mysql.New(mysql.Config{Conn: sqlDB}), &gorm.Config{})
	if err != nil {
		panic(err)
	}
}

func initConfig() {
	client.InitPath("conf/seatago.yml")
	initDB()
}

func main() {
	initConfig()
	ctx := context.Background()

	tccServiceProxy, err := tcc.NewTCCServiceProxy(&OrderTCCService{})
	if err != nil {
		log.Fatal(err)
	}

	// ---------------- Insert ----------------
	order := getData()

	err = tm.WithGlobalTx(ctx, &tm.GtxConfig{Name: "TCC_Insert"}, func(txCtx context.Context) error {
		ok, err := tccServiceProxy.Prepare(txCtx, order)
		if err != nil {
			return err
		}
		if okBool, ok := ok.(bool); !ok || !okBool {
			return errors.New("prepare insert failed")
		}
		return nil
	})
	if err != nil {
		log.Fatalf("insert transaction failed: %v", err)
	}
	log.Println("Insert success")

	// ---------------- Read ----------------
	var result OrderTblModel
	if err := gormDB.WithContext(ctx).
		Table("order_tbl").
		Where("user_id = ? AND commodity_code = ?", order.UserId, order.CommodityCode).
		First(&result).Error; err != nil {
		log.Fatal(err)
	}
	log.Printf("Read success Found: %+v", result)

	// ---------------- Update ----------------
	err = tm.WithGlobalTx(ctx, &tm.GtxConfig{Name: "TCC_Update"}, func(txCtx context.Context) error {
		if err := gormDB.WithContext(txCtx).
			Table("order_tbl").
			Where("id = ?", result.Id).
			Update("descs", "TCC update test").Error; err != nil {
			return err
		}
		log.Printf("[Update] id=%d descs=updated", result.Id)
		return nil
	})
	if err != nil {
		log.Fatalf("update transaction failed: %v", err)
	}
	log.Println("Update success")

	// ---------------- Delete ----------------
	err = tm.WithGlobalTx(ctx, &tm.GtxConfig{Name: "TCC_Delete"}, func(txCtx context.Context) error {
		if err := gormDB.WithContext(txCtx).
			Table("order_tbl").
			Where("id = ?", result.Id).
			Delete(nil).Error; err != nil {
			return err
		}
		log.Printf("[Delete] id=%d", result.Id)
		return nil
	})
	if err != nil {
		log.Fatalf("delete transaction failed: %v", err)
	}
	log.Println("Delete success")

	log.Println("TCC CRUD integration test passed! 🎉")

}

func getData() OrderTblModel {
	return OrderTblModel{
		Id:            20001,
		UserId:        "NO-100003",
		CommodityCode: "C100001",
		Count:         1,
		Money:         50,
		Descs:         "TCC insert test",
	}
}
