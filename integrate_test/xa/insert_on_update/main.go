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
	"fmt"
	"log"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"seata.apache.org/seata-go-samples/util"
	"seata.apache.org/seata-go/pkg/client"
	"seata.apache.org/seata-go/pkg/tm"
)

type OrderTblModel struct {
	Id            int64          `gorm:"column:id" json:"id"`
	UserId        string         `gorm:"column:user_id" json:"user_id"`
	CommodityCode string         `gorm:"commodity_code" json:"commodity_code"`
	Count         int64          `gorm:"count" json:"count"`
	Money         sql.NullInt64  `gorm:"money" json:"money"`
	Descs         string         `gorm:"descs" json:"descs"`
}

func main() {
	initConfig()

	ctx := context.Background()

	// prepare: insert initial data
	err := prepareData(ctx)
	if err != nil {
		log.Fatalf("failed to prepare data: %v", err)
		return
	}

	// test: insert on duplicate key update
	err = tm.WithGlobalTx(context.Background(), &tm.GtxConfig{
		Name:    "XASampleLocalGlobalTx_InsertOnUpdate",
		Timeout: time.Second * 30,
	}, insertOnUpdateData)

	if err != nil {
		log.Fatalf("failed to init transaction: %v", err)
		return
	}

	// check
	if checkData(ctx) != nil {
		panic("failed")
	}

	// XA mode doesn't use undo_log, so we don't need to check it
	log.Println("XA insert_on_update integration test passed successfully")
}

func initConfig() {
	client.InitPath("./conf/seatago.yml")
	initDB()
}

var gormDB *gorm.DB

func initDB() {
	sqlDB := util.GetXAMySqlDb()

	var err error
	gormDB, err = gorm.Open(mysql.New(mysql.Config{
		Conn: sqlDB,
	}), &gorm.Config{})
	if err != nil {
		panic("open DB error")
	}
}

func getPrepareData() OrderTblModel {
	return OrderTblModel{
		Id:            200,
		UserId:        "NO-100005",
		CommodityCode: "C100003",
		Count:         30,
		Money:         sql.NullInt64{Int64: 300, Valid: true},
		Descs:         "init desc",
	}
}

func getUpdatedDescs() string {
	return fmt.Sprintf("insert on update descs %d", time.Now().Unix())
}

// prepareData insert initial test data
func prepareData(ctx context.Context) error {
	// delete old test data if exists
	gormDB.WithContext(ctx).Table("order_tbl").Where("id = ?", 200).Delete(&OrderTblModel{})

	data := getPrepareData()
	return gormDB.WithContext(ctx).Table("order_tbl").Create(&data).Error
}

// insertOnUpdateData insert on duplicate key update operation
func insertOnUpdateData(ctx context.Context) error {
	newDescs := getUpdatedDescs()
	sql := "insert into order_tbl (id, user_id, commodity_code, count, money, descs) values (?, ?, ?, ?, ?, ?) " +
		"on duplicate key update descs=?"
	
	sqlDB, err := gormDB.DB()
	if err != nil {
		return err
	}
	
	_, err = sqlDB.ExecContext(ctx, sql, 200, "NO-100005", "C100003", 30, 300, "init desc", newDescs)
	return err
}

func checkData(ctx context.Context) error {
	var order OrderTblModel
	err := gormDB.WithContext(ctx).Table("order_tbl").Where("id = ?", 200).First(&order).Error
	if err != nil {
		return err
	}
	
	// check if the description was updated (not the original)
	if order.Descs == "init desc" {
		return fmt.Errorf("check data failed: data was not updated by INSERT ON DUPLICATE KEY UPDATE")
	}
	
	// check other fields remain unchanged
	prepareData := getPrepareData()
	if order.UserId != prepareData.UserId || 
		order.CommodityCode != prepareData.CommodityCode ||
		order.Count != prepareData.Count ||
		order.Money.Int64 != prepareData.Money.Int64 {
		return fmt.Errorf("check data failed: unexpected data changes")
	}
	
	return nil
}

