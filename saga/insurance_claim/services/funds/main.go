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
		log.Fatalf("failed to open the database: %v", err)
	}
	defer db.Close()

	if err := app.EnsureBusinessSchema(db); err != nil {
		log.Fatalf("failed to initialize the business schema: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		httpjson.WriteText(w, http.StatusOK, "OK")
	})
	mux.HandleFunc("/ReservePayoutFunds", func(w http.ResponseWriter, r *http.Request) {
		args, err := httpjson.ReadArgs(r, 3)
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
		amount, err := httpjson.IntArg(args, 2)
		if err != nil {
			httpjson.WriteText(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := app.ReserveFunds(db, businessKey, claimID, amount); err != nil {
			httpjson.WriteText(w, http.StatusInternalServerError, err.Error())
			return
		}
		log.Printf("operation=ReservePayoutFunds businessKey=%s claimId=%s amount=%d status=SUCCESS", businessKey, claimID, amount)
		httpjson.WriteText(w, http.StatusOK, "FUNDS_RESERVED")
	})
	mux.HandleFunc("/ReleasePayoutFunds", func(w http.ResponseWriter, r *http.Request) {
		args, err := httpjson.ReadArgs(r, 2)
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
		if err := app.ReleaseFunds(db, businessKey, claimID); err != nil {
			httpjson.WriteText(w, http.StatusInternalServerError, err.Error())
			return
		}
		log.Printf("operation=ReleasePayoutFunds businessKey=%s claimId=%s status=SUCCESS", businessKey, claimID)
		httpjson.WriteText(w, http.StatusOK, "FUNDS_RELEASED")
	})

	addr := fmt.Sprintf(":%s", settings.FundsPort)
	log.Printf("funds service listening on %s\n", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
