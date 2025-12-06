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
	"gorm.io/gorm/clause"

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

type TCCInsertOnUpdateService struct{}

func (t *TCCInsertOnUpdateService) Prepare(ctx context.Context, params interface{}) (bool, error) {
	order := params.(OrderTblModel)

	// Insert on update operation using GORM
	err := gormDB.WithContext(ctx).Table("order_tbl").Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}}, // by primary key
		DoUpdates: clause.Assignments(map[string]interface{}{"descs": order.Descs}),
	}).Create(&order).Error

	if err != nil {
		return false, err
	}
	log.Printf("[Prepare] insert on update order %+v", order)
	return true, nil
}

func (t *TCCInsertOnUpdateService) Commit(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) {
	log.Printf("[Commit] confirm insert on update order")
	return true, nil
}

func (t *TCCInsertOnUpdateService) Rollback(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) {
	log.Printf("[Rollback] cancel insert on update")

	// Delete the record that was inserted/updated in Prepare phase
	// Using the same ID that was used in the test
	if err := gormDB.WithContext(ctx).
		Table("order_tbl").
		Where("id = ?", 1). // The test uses ID=1
		Delete(nil).Error; err != nil {
		return false, err
	}
	log.Printf("[Rollback] deleted order with id=1")
	return true, nil
}

func (t *TCCInsertOnUpdateService) GetActionName() string {
	return "TCCInsertOnUpdateService"
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

	tccServiceProxy, err := tcc.NewTCCServiceProxy(&TCCInsertOnUpdateService{})
	if err != nil {
		log.Fatal(err)
	}

	// ---------------- Insert On Update ----------------
	order := getData()

	err = tm.WithGlobalTx(ctx, &tm.GtxConfig{Name: "TCC_InsertOnUpdate"}, func(txCtx context.Context) error {
		ok, err := tccServiceProxy.Prepare(txCtx, order)
		if err != nil {
			return err
		}
		if okBool, ok := ok.(bool); !ok || !okBool {
			return errors.New("prepare insert on update failed")
		}
		return nil
	})
	if err != nil {
		log.Fatalf("insert on update transaction failed: %v", err)
	}
	log.Println("Insert on update success")

	// ---------------- Verify ----------------
	var result OrderTblModel
	if err := gormDB.WithContext(ctx).
		Table("order_tbl").
		Where("id = ?", order.Id).
		First(&result).Error; err != nil {
		log.Fatal(err)
	}
	log.Printf("Verify success Found: %+v", result)

	log.Println("TCC insert on update integration test passed! 🎉")
}

func getData() OrderTblModel {
	return OrderTblModel{
		Id:            1,
		UserId:        "NO-100003",
		CommodityCode: "C100001",
		Count:         101,
		Money:         11,
		Descs:         "TCC insert on update test",
	}
}
