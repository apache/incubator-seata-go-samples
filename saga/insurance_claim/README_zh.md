<!--
  ~ Licensed to the Apache Software Foundation (ASF) under one or more
  ~ contributor license agreements.  See the NOTICE file distributed with
  ~ this work for additional information regarding copyright ownership.
  ~ The ASF licenses this file to You under the Apache License, Version 2.0
  ~ (the "License"); you may not use this file except in compliance with
  ~ the License.  You may obtain a copy of the License at
  ~
  ~     http://www.apache.org/licenses/LICENSE-2.0
  ~
  ~ Unless required by applicable law or agreed to in writing, software
  ~ distributed under the License is distributed on an "AS IS" BASIS,
  ~ WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  ~ See the License for the specific language governing permissions and
  ~ limitations under the License.
-->

# Insurance Claim Saga Sample

本示例演示一个遗留的长流程理赔处理如何迁移到 Seata Go Saga。

场景步骤如下：

1. 校验理赔人身份
2. 创建定损记录
3. 预留赔付资金
4. 通知定损员
5. 执行银行打款

遗留实现是一个串行流程。前 4 步成功后，如果第 5 步银行打款失败，系统会留下已校验身份、已创建定损单、已预留资金、已通知定损员等中间状态，无法自动回滚。

Saga 迁移后，每个步骤都会拆成前向动作和补偿动作：

| 前向动作 | 补偿动作 |
| --- | --- |
| VerifyIdentity | UnverifyClaim |
| CreateDamageAssessment | DeleteDamageAssessment |
| ReservePayoutFunds | ReleasePayoutFunds |
| NotifyAssignedSurveyor | CancelSurveyorNotification |
| ExecuteBankTransfer | 无 |

当 `ExecuteBankTransfer` 失败时，Saga 会按逆序执行：

1. CancelSurveyorNotification
2. ReleasePayoutFunds
3. DeleteDamageAssessment
4. UnverifyClaim

## 目录说明

- `legacy/`：遗留串行版本，对照“迁移前”的问题
- `orchestrator/`：Saga 编排启动器
- `services/`：五个独立 Go HTTP 服务
- `statelang/insurance_claim_saga.json`：Saga 状态机定义
- `sql/mysql_claim_saga_schema.sql`：Saga 持久化表 + 业务表示例
- `docker-compose.yml`：MySQL 与 Seata Server

## 启动基础设施

```bash
cd saga/insurance_claim
docker-compose up -d
```

默认端口：

- MySQL: `3306`
- Seata Server: `8091`
- identity service: `18081`
- assessment service: `18082`
- funds service: `18083`
- surveyor service: `18084`
- transfer service: `18085`

## 启动五个服务

在仓库根目录分别打开 5 个终端：

```bash
go run ./saga/insurance_claim/services/identity
go run ./saga/insurance_claim/services/assessment
go run ./saga/insurance_claim/services/funds
go run ./saga/insurance_claim/services/surveyor
go run ./saga/insurance_claim/services/transfer
```

也可以在 `saga/insurance_claim` 目录内运行，把命令里的 `saga/insurance_claim/` 前缀去掉即可，例如 `go run ./orchestrator`。

## 运行遗留串行流程

成功场景：

```bash
go run ./saga/insurance_claim/legacy
```

模拟银行打款失败：

```bash
go run ./saga/insurance_claim/legacy -failTransfer
```

失败时会直接停止，并打印当前业务快照。你会看到前 4 步留下的中间状态仍然存在，这正是遗留实现的问题。

## 运行 Saga 迁移后的流程

成功场景：

```bash
go run ./saga/insurance_claim/orchestrator
```

模拟银行打款失败：

```bash
go run ./saga/insurance_claim/orchestrator -failTransfer
```

失败场景下会输出：

- `xid`
- Saga 最终状态
- compensation status
- 业务快照
- `actionTrail`

重点看 `actionTrail`，可以看到：

1. 先执行 4 个前向动作
2. 打款失败
3. 再按逆序执行 4 个补偿动作

## 关键实现说明

- 五个服务都是独立进程，使用 HTTP 暴露前向动作和补偿动作
- Saga 编排器通过 seata-go 内置 `http` invoker 调用这些服务
- 状态机使用 `CompensateState` 描述每个前向动作对应的补偿动作
- `failTransfer` 参数用于稳定复现银行打款失败
- `claim_step_log` 会记录前向和补偿执行顺序，便于观察迁移效果
- MySQL 可通过 `MYSQL_HOST`、`MYSQL_PORT`、`MYSQL_USERNAME`、`MYSQL_PASSWORD`、`MYSQL_DB` 覆盖；同时兼容 `MYSQL_USER` 和 `MYSQL_PWD`

## 使用本地 seata-go 调试

如果要配合本地 `../incubator-seata-go` 调试：

```bash
go mod edit -replace seata.apache.org/seata-go=../incubator-seata-go
go run ./saga/insurance_claim/orchestrator -failTransfer
go mod edit -dropreplace seata.apache.org/seata-go
```
