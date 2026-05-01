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
	"fmt"
	"log"
	"net/http"

	"seata.apache.org/seata-go-samples/saga/insurance_claim/internal/app"
	"seata.apache.org/seata-go-samples/saga/insurance_claim/internal/httpjson"
)

func main() {
	settings := app.LoadSettings()
	db, err := app.OpenDB()
	if err != nil {
		log.Fatalf("打开数据库失败: %v", err)
	}
	defer db.Close()

	if err := app.EnsureBusinessSchema(db); err != nil {
		log.Fatalf("初始化业务表失败: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		httpjson.WriteText(w, http.StatusOK, "OK")
	})
	mux.HandleFunc("/ExecuteBankTransfer", func(w http.ResponseWriter, r *http.Request) {
		args, err := httpjson.ReadArgs(r, 5)
		if err != nil {
			httpjson.WriteText(w, http.StatusBadRequest, err.Error())
			return
		}
		businessKey, err := httpjson.StringArg(args, 0)
		if err != nil {
			httpjson.WriteText(w, http.StatusBadRequest, err.Error())
			return
		}
		claimID, err := httpjson.StringArg(args, 1)
		if err != nil {
			httpjson.WriteText(w, http.StatusBadRequest, err.Error())
			return
		}
		bankAccount, err := httpjson.StringArg(args, 2)
		if err != nil {
			httpjson.WriteText(w, http.StatusBadRequest, err.Error())
			return
		}
		amount, err := httpjson.IntArg(args, 3)
		if err != nil {
			httpjson.WriteText(w, http.StatusBadRequest, err.Error())
			return
		}
		failTransfer, err := httpjson.BoolArg(args, 4)
		if err != nil {
			httpjson.WriteText(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := app.ExecuteTransfer(db, businessKey, claimID, bankAccount, amount, failTransfer); err != nil {
			httpjson.WriteText(w, http.StatusInternalServerError, err.Error())
			return
		}
		httpjson.WriteText(w, http.StatusOK, "BANK_TRANSFER_SUCCESS")
	})

	addr := fmt.Sprintf(":%s", settings.TransferPort)
	log.Printf("transfer service listening on %s\n", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
