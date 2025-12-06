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
	Id            int64  `gorm:"column:id" json:"id"`
	UserId        string `gorm:"column:user_id" json:"user_id"`
	CommodityCode string `gorm:"commodity_code" json:"commodity_code"`
	Count         int64  `gorm:"count" json:"count"`
	Money         int64  `gorm:"money" json:"money"`
	Descs         string `gorm:"descs" json:"descs"`
}

func main() {
	initConfig()

	ctx := context.Background()

	err := prepareData(ctx)
	if err != nil {
		log.Fatalf("failed to prepare data: %v", err)
		return
	}

	err = tm.WithGlobalTx(context.Background(), &tm.GtxConfig{
		Name:    "XASampleLocalGlobalTx_SelectForUpdate",
		Timeout: time.Second * 30,
	}, selectForUpdateData)

	if err != nil {
		log.Fatalf("failed to init transaction: %v", err)
		return
	}

	if checkData(ctx) != nil {
		panic("failed")
	}

	log.Println("XA select_for_update integration test passed successfully")
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
		Id:            300,
		UserId:        "NO-100006",
		CommodityCode: "C100004",
		Count:         20,
		Money:         200,
		Descs:         "select for update test",
	}
}

func getUpdatedDescs() string {
	return fmt.Sprintf("updated after select for update %d", time.Now().Unix())
}

func prepareData(ctx context.Context) error {
	gormDB.WithContext(ctx).Table("order_tbl").Where("id = ?", 300).Delete(&OrderTblModel{})

	data := getPrepareData()
	return gormDB.WithContext(ctx).Table("order_tbl").Create(&data).Error
}

func selectForUpdateData(ctx context.Context) error {
	sqlDB, err := gormDB.DB()
	if err != nil {
		return err
	}

	rows, err := sqlDB.QueryContext(ctx, "SELECT id, user_id, descs FROM order_tbl WHERE id=? FOR UPDATE", 300)
	if err != nil {
		return fmt.Errorf("select for update failed: %v", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return fmt.Errorf("no record found with id=300")
	}

	var id int64
	var userId, descs string
	err = rows.Scan(&id, &userId, &descs)
	if err != nil {
		return fmt.Errorf("scan failed: %v", err)
	}
	rows.Close()

	log.Printf("Selected record: id=%d, user_id=%s, descs=%s", id, userId, descs)

	newDescs := getUpdatedDescs()
	err = gormDB.WithContext(ctx).Table("order_tbl").
		Where("id = ?", 300).
		Update("descs", newDescs).Error

	if err != nil {
		return fmt.Errorf("update after select for update failed: %v", err)
	}

	return nil
}

func checkData(ctx context.Context) error {
	var order OrderTblModel
	err := gormDB.WithContext(ctx).Table("order_tbl").Where("id = ?", 300).First(&order).Error
	if err != nil {
		return err
	}

	if order.Descs == "select for update test" {
		return fmt.Errorf("check data failed: data was not updated after select for update")
	}

	prepareData := getPrepareData()
	if order.UserId != prepareData.UserId ||
		order.CommodityCode != prepareData.CommodityCode ||
		order.Count != prepareData.Count ||
		order.Money != prepareData.Money {
		return fmt.Errorf("check data failed: unexpected data changes")
	}

	log.Printf("Verification successful: descs updated to '%s'", order.Descs)
	return nil
}
