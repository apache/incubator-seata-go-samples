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
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"

	"seata.apache.org/seata-go-samples/saga/insurance_claim/internal/app"
)

func main() {
	var (
		businessKey  string
		claimID      string
		claimantID   string
		assessmentID string
		surveyorID   string
		bankAccount  string
		payoutAmount int
		failTransfer bool
	)

	flag.StringVar(&businessKey, "businessKey", "insurance-claim-legacy-demo", "业务主键")
	flag.StringVar(&claimID, "claimId", "claim-1001", "理赔单号")
	flag.StringVar(&claimantID, "claimantId", "claimant-9001", "理赔人编号")
	flag.StringVar(&assessmentID, "assessmentId", "assessment-7001", "定损单号")
	flag.StringVar(&surveyorID, "surveyorId", "surveyor-3001", "定损员编号")
	flag.StringVar(&bankAccount, "bankAccount", "6222020202020202", "收款银行卡")
	flag.IntVar(&payoutAmount, "payoutAmount", 1500, "打款金额")
	flag.BoolVar(&failTransfer, "failTransfer", false, "是否模拟银行打款失败")
	flag.Parse()

	settings := app.LoadSettings()
	db, err := app.OpenDB()
	if err != nil {
		fmt.Fprintf(os.Stderr, "打开数据库失败: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := app.EnsureBusinessSchema(db); err != nil {
		fmt.Fprintf(os.Stderr, "初始化业务表失败: %v\n", err)
		os.Exit(1)
	}
	if err := app.ResetClaimData(db, businessKey, claimID); err != nil {
		fmt.Fprintf(os.Stderr, "重置示例数据失败: %v\n", err)
		os.Exit(1)
	}

	steps := []struct {
		name string
		url  string
		args []any
	}{
		{"VerifyIdentity", settings.IdentityBaseURL() + "VerifyIdentity", []any{businessKey, claimID, claimantID}},
		{"CreateDamageAssessment", settings.AssessmentBaseURL() + "CreateDamageAssessment", []any{businessKey, claimID, assessmentID}},
		{"ReservePayoutFunds", settings.FundsBaseURL() + "ReservePayoutFunds", []any{businessKey, claimID, payoutAmount}},
		{"NotifyAssignedSurveyor", settings.SurveyorBaseURL() + "NotifyAssignedSurveyor", []any{businessKey, claimID, surveyorID}},
		{"ExecuteBankTransfer", settings.TransferBaseURL() + "ExecuteBankTransfer", []any{businessKey, claimID, bankAccount, payoutAmount, failTransfer}},
	}

	for _, step := range steps {
		if err := invoke(step.url, step.args); err != nil {
			fmt.Printf("mode=legacy businessKey=%s failedStep=%s err=%v\n", businessKey, step.name, err)
			snapshot, snapErr := app.LoadSnapshot(db, claimID)
			if snapErr != nil {
				fmt.Fprintf(os.Stderr, "读取理赔快照失败: %v\n", snapErr)
				os.Exit(1)
			}
			fmt.Println(app.FormatSnapshot(snapshot))
			os.Exit(1)
		}
	}

	snapshot, err := app.LoadSnapshot(db, claimID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "读取理赔快照失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("mode=legacy businessKey=%s status=SUCCESS\n", businessKey)
	fmt.Println(app.FormatSnapshot(snapshot))
}

func invoke(url string, args []any) error {
	body, err := json.Marshal(args)
	if err != nil {
		return err
	}
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("%s", string(raw))
	}
	return nil
}
