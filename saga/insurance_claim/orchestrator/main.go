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
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"seata.apache.org/seata-go-samples/saga/insurance_claim/internal/app"
	"seata.apache.org/seata-go/pkg/client"
	engcfg "seata.apache.org/seata-go/pkg/saga/statemachine/engine/config"
	"seata.apache.org/seata-go/pkg/saga/statemachine/engine/core"
	"seata.apache.org/seata-go/pkg/saga/statemachine/engine/invoker"
)

func main() {
	var (
		seataConf    string
		engineConf   string
		businessKey  string
		claimID      string
		claimantID   string
		assessmentID string
		surveyorID   string
		bankAccount  string
		payoutAmount int
		failTransfer bool
	)

	flag.StringVar(&seataConf, "seataConf", "saga/insurance_claim/seatago.yaml", "seata-go 客户端配置路径")
	flag.StringVar(&engineConf, "engineConf", "saga/insurance_claim/config.yaml", "Saga 引擎配置路径")
	flag.StringVar(&businessKey, "businessKey", "insurance-claim-saga-demo", "业务主键")
	flag.StringVar(&claimID, "claimId", "claim-1001", "理赔单号")
	flag.StringVar(&claimantID, "claimantId", "claimant-9001", "理赔人编号")
	flag.StringVar(&assessmentID, "assessmentId", "assessment-7001", "定损单号")
	flag.StringVar(&surveyorID, "surveyorId", "surveyor-3001", "定损员编号")
	flag.StringVar(&bankAccount, "bankAccount", "6222020202020202", "收款银行卡")
	flag.IntVar(&payoutAmount, "payoutAmount", 1500, "打款金额")
	flag.BoolVar(&failTransfer, "failTransfer", false, "是否模拟银行打款失败")
	flag.Parse()

	client.InitPath(seataConf)

	engine, err := core.NewProcessCtrlStateMachineEngine()
	if err != nil {
		fmt.Fprintf(os.Stderr, "创建状态机引擎失败: %v\n", err)
		os.Exit(1)
	}

	cfgIface := engine.GetStateMachineConfig()
	cfg, ok := cfgIface.(*engcfg.DefaultStateMachineConfig)
	if !ok {
		fmt.Fprintf(os.Stderr, "未知状态机配置类型: %T\n", cfgIface)
		os.Exit(1)
	}

	if err := cfg.LoadConfig(engineConf); err != nil {
		fmt.Fprintf(os.Stderr, "加载引擎配置失败: %v\n", err)
		os.Exit(1)
	}
	if wd, err := os.Getwd(); err == nil {
		absPattern := filepath.Join(wd, "saga/insurance_claim/statelang/*.json")
		_ = cfg.RegisterStateMachineDef([]string{absPattern})
	}
	if err := cfg.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "初始化引擎配置失败: %v\n", err)
		os.Exit(1)
	}

	settings := app.LoadSettings()
	registerHTTPClients(cfgIface, settings)

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

	params := map[string]any{
		"businessKey":  businessKey,
		"claimId":      claimID,
		"claimantId":   claimantID,
		"assessmentId": assessmentID,
		"surveyorId":   surveyorID,
		"bankAccount":  bankAccount,
		"payoutAmount": payoutAmount,
		"failTransfer": failTransfer,
	}

	instance, err := engine.Start(context.Background(), "InsuranceClaimSaga", businessKey, params)
	if err != nil {
		fmt.Fprintf(os.Stderr, "启动 Saga 失败: %v\n", err)
		os.Exit(1)
	}

	snapshot, err := app.LoadSnapshot(db, claimID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "读取理赔快照失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("mode=saga businessKey=%s xid=%s status=%s compensationStatus=%s\n",
		businessKey, instance.ID(), instance.Status(), instance.CompensationStatus())
	fmt.Println(app.FormatSnapshot(snapshot))
}

func registerHTTPClients(cfgIface any, settings app.Settings) {
	cfg := cfgIface.(*engcfg.DefaultStateMachineConfig)
	httpInvoker, ok := cfg.ServiceInvokerManager().ServiceInvoker("http").(*invoker.HTTPInvoker)
	if !ok {
		panic("http invoker 未初始化")
	}

	clientConfig := &http.Client{}
	httpInvoker.RegisterClient("identityService", invoker.NewHTTPClient("identityService", settings.IdentityBaseURL(), clientConfig))
	httpInvoker.RegisterClient("assessmentService", invoker.NewHTTPClient("assessmentService", settings.AssessmentBaseURL(), clientConfig))
	httpInvoker.RegisterClient("fundsService", invoker.NewHTTPClient("fundsService", settings.FundsBaseURL(), clientConfig))
	httpInvoker.RegisterClient("surveyorService", invoker.NewHTTPClient("surveyorService", settings.SurveyorBaseURL(), clientConfig))
	httpInvoker.RegisterClient("transferService", invoker.NewHTTPClient("transferService", settings.TransferBaseURL(), clientConfig))
}
