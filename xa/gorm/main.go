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
	"time"

	"github.com/seata/seata-go/pkg/client"
	sql2 "github.com/seata/seata-go/pkg/datasource/sql"
	"github.com/seata/seata-go/pkg/tm"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
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
	// insert
	tm.WithGlobalTx(context.Background(), &tm.GtxConfig{
		Name:    "ATSampleLocalGlobalTx",
		Timeout: time.Second * 30,
	}, insertData)
	<-make(chan struct{})
}

func initConfig() {
	// init seata client config
	client.InitPath("/Users/rain/go/src/wang1309/seata-go-samples/conf/seatago.yml")
	// init db object
	initDB()
}

var gormDB *gorm.DB

func initDB() {
	sqlDB, err := sql.Open(sql2.SeataXAMySQLDriver, "root:12345678@tcp(127.0.0.1:3306)/seata_client?multiStatements=true&interpolateParams=true")
	if err != nil {
		panic("init service error")
	}

	gormDB, err = gorm.Open(mysql.New(mysql.Config{
		Conn: sqlDB,
	}), &gorm.Config{})
}

// insertData insert one data
func insertData(ctx context.Context) error {
	data := OrderTblModel{
		Id:            1,
		UserId:        "NO-100003",
		CommodityCode: "C100001",
		Count:         101,
		Money:         11,
		Descs:         "insert desc",
	}

	return gormDB.WithContext(ctx).Table("order_tbl").Create(&data).Error
}

// deleteData delete one data
func deleteData(ctx context.Context) error {
	return gormDB.WithContext(ctx).Where("id = ?", "1").Delete(&OrderTblModel{}).Error
}

// updateDate update one data
func updateData(ctx context.Context) error {
	return gormDB.WithContext(ctx).Model(&OrderTblModel{}).Where("id = ?", "1").Update("commodity_code", "C100002").Error
}
