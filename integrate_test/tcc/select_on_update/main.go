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

type TCCSelectForUpdateService struct{}

func (t *TCCSelectForUpdateService) Prepare(ctx context.Context, params interface{}) (bool, error) {
	queryParams := params.(map[string]interface{})
	userId := queryParams["userId"].(string)
	commodityCode := queryParams["commodityCode"].(string)

	// Select for update operation using GORM
	var order OrderTblModel
	err := gormDB.WithContext(ctx).Table("order_tbl").
		Where("user_id = ? AND commodity_code = ?", userId, commodityCode).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&order).Error

	if err != nil {
		return false, err
	}
	log.Printf("[Prepare] select for update found order %+v", order)
	return true, nil
}

func (t *TCCSelectForUpdateService) Commit(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) {
	log.Printf("[Commit] confirm select for update")
	return true, nil
}

func (t *TCCSelectForUpdateService) Rollback(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) {
	log.Printf("[Rollback] cancel select for update")
	// Select for update doesn't modify data, just releases the lock
	// The database will automatically release the lock when transaction ends
	return true, nil
}

func (t *TCCSelectForUpdateService) GetActionName() string {
	return "TCCSelectForUpdateService"
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

	tccServiceProxy, err := tcc.NewTCCServiceProxy(&TCCSelectForUpdateService{})
	if err != nil {
		log.Fatal(err)
	}

	// ---------------- Select For Update ----------------
	queryParams := getData()

	err = tm.WithGlobalTx(ctx, &tm.GtxConfig{Name: "TCC_SelectForUpdate"}, func(txCtx context.Context) error {
		ok, err := tccServiceProxy.Prepare(txCtx, queryParams)
		if err != nil {
			return err
		}
		if okBool, ok := ok.(bool); !ok || !okBool {
			return errors.New("prepare select for update failed")
		}
		return nil
	})
	if err != nil {
		log.Fatalf("select for update transaction failed: %v", err)
	}
	log.Println("Select for update success")

	log.Println("TCC select for update integration test passed! 🎉")
}

func getData() map[string]interface{} {
	return map[string]interface{}{
		"userId":        "NO-100001",
		"commodityCode": "C100000",
	}
}
