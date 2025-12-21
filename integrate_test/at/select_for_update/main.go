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
	"seata.apache.org/seata-go/pkg/client"
	sql2 "seata.apache.org/seata-go/pkg/datasource/sql"
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

	// test: select for update
	err := tm.WithGlobalTx(context.Background(), &tm.GtxConfig{
		Name:    "ATSampleLocalGlobalTx_SelectForUpdate",
		Timeout: time.Second * 30,
	}, selectForUpdateData)

	if err != nil {
		log.Fatalf("failed to init transaction: %v", err)
		return
	}

	ctx := context.Background()

	if err := checkData(ctx); err != nil {
		log.Fatalf("check data failed: %v", err)
	}

	time.Sleep(time.Second * 10)
	if err := checkUndoLogData(ctx); err != nil {
		log.Fatalf("check undo log failed: %v", err)
	}

	log.Println("select for update test passed successfully")
}

func initConfig() {
	client.InitPath("./conf/seatago.yml")
	initDB()
}

var gormDB *gorm.DB
var sqlDB *sql.DB

func initDB() {
	var err error
	sqlDB, err = sql.Open(sql2.SeataATMySQLDriver, "root:12345678@tcp(127.0.0.1:3306)/seata_client?multiStatements=true&interpolateParams=true")
	if err != nil {
		panic("init service error")
	}

	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(time.Minute * 3)

	gormDB, err = gorm.Open(mysql.New(mysql.Config{
		Conn: sqlDB,
	}), &gorm.Config{})
	if err != nil {
		panic("open DB error")
	}
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

// selectForUpdateData select for update
func selectForUpdateData(ctx context.Context) error {
	selectSQL := "SELECT id, user_id, commodity_code, count, money, descs FROM order_tbl WHERE id = ? FOR UPDATE"
	_, err := sqlDB.ExecContext(ctx, selectSQL, 1)
	if err != nil {
		return fmt.Errorf("select for update failed: %w", err)
	}

	updateSQL := "UPDATE order_tbl SET descs = ?, count = count + 10 WHERE id = ?"
	_, err = sqlDB.ExecContext(ctx, updateSQL, "updated by select for update", 1)
	if err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	return nil
}

func checkData(ctx context.Context) error {
	var data OrderTblModel
	err := gormDB.WithContext(ctx).Table("order_tbl").Where("id = ?", 1).Find(&data).Error
	if err != nil {
		return err
	}
	if data.Descs != "updated by select for update" {
		return fmt.Errorf("check data failed: expected descs='updated by select for update', got '%s'", data.Descs)
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
