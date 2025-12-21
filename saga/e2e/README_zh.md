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

# Saga E2E（MySQL）— 端到端使用说明

本示例在 `saga/e2e` 目录下，提供一键脚本与 DB 校验工具。

## 一键运行（推荐）

前置条件：本机需安装 Go 1.20+、Docker 与 docker-compose。

使用 docker‑compose 重建/启动 MySQL 与 Seata Server，等待就绪后依次运行 3 个场景并进行 DB 校验：

```
saga/e2e/run_all.sh --up \
  --seata saga/e2e/seatago.yaml \
  --engine saga/e2e/config.yaml
```

## 使用本地 seata-go 进行调试

sample 仓库当前依赖发布版的 seata-go。如需在调试时直接复用
`../incubator-seata-go` 的最新代码，可在运行前临时添加 replace：

```
go mod edit -replace seata.apache.org/seata-go=../incubator-seata-go
go mod tidy # 依赖有变更时执行
```

执行完成后建议移除 replace，保持 go.mod 干净：

```
go mod edit -dropreplace seata.apache.org/seata-go
```

如果使用 Go 1.18+，也可以临时创建 workspace：

```
go work init
go work use .
go work use ../incubator-seata-go
go run ./saga/e2e
rm go.work
```

说明：
- 脚本会 `docker-compose down -v` 清理旧容器与卷，再 `up -d --force-recreate` 全新拉起
- 自动等待 MySQL、Seata Server 端口就绪，并额外等待一段时间，避免刚启动时 MySQL 的 EOF 抖动
- 依次运行 3 个场景并进行 DB 校验：
  - success：前向执行成功
  - compensate-balance：余额扣减失败触发补偿，最终 Fail（补偿 SU）
  - compensate-inventory：库存首步失败，无补偿，最终 Fail

可调参数（环境变量）：
- `WAIT_TIMEOUT`（默认 60）：端口等待超时秒数
- `WAIT_INTERVAL`（默认 2）：端口轮询间隔秒数
- `WAIT_READY_MARGIN`（默认 15）：端口可达后的额外等待秒数

预期输出：
- `DB validation (success) OK`
- `DB validation (compensate-balance) OK`
- `DB validation (compensate-inventory) OK`
- `[+] All e2e scenarios finished`

## 配置

- Seata 客户端（`seatago.yaml`）
  - `application-id`、`tx-service-group`
  - `service.grouplist.default`：Seata Server 地址（`ip:port`）

- Saga 引擎（`config.yaml`）
  - `store_enabled: true`、`store_type: mysql`
  - `store_dsn: user:pass@tcp(127.0.0.1:3306)/seata_saga?parseTime=true`
  - `tc_enabled: true`
  - `state_machine_resources: [saga/e2e/statelang/*.json]`

## 单场景运行

- 启动并运行（重建容器）：`saga/e2e/up_and_run.sh`
- 仅运行成功场景：`saga/e2e/run.sh [seatago.yaml] [config.yaml]`
- 运行补偿场景：`saga/e2e/run_compensation.sh [seatago.yaml] [config.yaml] [compensate-balance|compensate-inventory]`

## DB 校验工具

用于在运行结束后对最终状态进行校验：

```
go run ./saga/e2e/dbcheck \
  -engine saga/e2e/config.yaml \
  -xid <XID> \
  -scenario success|compensate-balance|compensate-inventory
```

校验点包括：
- `seata_state_machine_inst`：最终 `status`、`compensation_status`、`is_running`、`gmt_end`
- `seata_state_inst`：关键前向/补偿状态与 `state_id_compensated_for` 关联

## 初始化数据库表（非 docker‑compose 场景）

```
MYSQL_HOST=127.0.0.1 MYSQL_PORT=3306 \
MYSQL_USER=root MYSQL_PWD=secret MYSQL_DB=seata_saga \
saga/e2e/migrate.sh
```

以上脚本会执行 `sql/mysql_saga_schema.sql` 创建相关表。

## 常见问题

- 无法连接 Seata Server：检查 `seatago.yaml` 的 `service.grouplist.default` 与网络连通性。
- BranchRegister 失败或 branchId 非法：确认资源标识 `applicationId#txServiceGroup` 与 TC 的 vgroup 映射一致。
- MySQL 刚启动时 EOF/“bad connection”：增大 `WAIT_READY_MARGIN`（如 20）。
- 成功场景“乐观锁 0 行更新”提示：幂等/并发下的正常现象，已降为 debug，不影响最终结果。
- TC GlobalReport 超时：仅记录日志，不影响本地最终态；DB 校验仍应通过（与 Java 行为一致）。

## 参考路径

- 状态机 JSON：`statelang/reduce_inventory_and_balance.json`
- 引擎/持久化关键路径：`pkg/saga/statemachine/engine/pcext/*`、`pkg/saga/statemachine/store/db/statelog.go`
- 脚本：`run_all.sh`、`up_and_run.sh`、`run.sh`、`run_compensation.sh`
