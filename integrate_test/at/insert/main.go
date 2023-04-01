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
	"github.com/seata/seata-go/pkg/client"
	sql2 "github.com/seata/seata-go/pkg/datasource/sql"
	"github.com/seata/seata-go/pkg/tm"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"time"
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

	// test: insert
	tm.WithGlobalTx(context.Background(), &tm.GtxConfig{
		Name:    "ATSampleLocalGlobalTx",
		Timeout: time.Second * 30,
	}, insertData)

	ctx := context.Background()

	// check
	if checkData(ctx) != nil {
		panic("failed")
	}

	// wait clean undo log
	time.Sleep(time.Second * 10)
	if checkUndoLogData(ctx) != nil {
		panic("failed")
	}
}

func initConfig() {
	client.InitPath("./conf/seatago.yml")
	initDB()
}

var gormDB *gorm.DB

func initDB() {
	sqlDB, err := sql.Open(sql2.SeataATMySQLDriver, "root:12345678@tcp(127.0.0.1:3306)/seata_client?multiStatements=true&interpolateParams=true")
	if err != nil {
		panic("init service error")
	}

	gormDB, err = gorm.Open(mysql.New(mysql.Config{
		Conn: sqlDB,
	}), &gorm.Config{})
}

func getData() OrderTblModel {
	return OrderTblModel{
		UserId:        "NO-100003",
		CommodityCode: "C100001",
		Count:         101,
		Money:         11,
		Descs:         "insert desc",
	}
}

// insertData insert one data
func insertData(ctx context.Context) error {
	data := getData()

	return gormDB.WithContext(ctx).Table("order_tbl").Create(&data).Error
}

func checkData(ctx context.Context) error {
	query := getData()
	var count int64
	err := gormDB.WithContext(ctx).Table("order_tbl").Where(query).Count(&count).Error
	if err != nil {
		return err
	}
	if count != 1 {
		return fmt.Errorf("check data failed")
	}
	return nil
}

func checkUndoLogData(ctx context.Context) error {
	var count int64
	err := gormDB.WithContext(ctx).Table("undo_log").Count(&count).Error
	if err != nil {
		return err
	}
	if count != 0 {
		return fmt.Errorf("check undolog failed")
	}
	return nil
}
