<!--
 Licensed to the Apache Software Foundation (ASF) under one or more
 contributor license agreements.  See the NOTICE file distributed with
 this work for additional information regarding copyright ownership.
 The ASF licenses this file to You under the Apache License, Version 2.0
 (the "License"); you may not use this file except in compliance with
 the License.  You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
-->

# XA Bank Transfer

This sample demonstrates an XA global transaction across two independent MySQL
databases.

- `transfer-service` starts the global transaction and calls the two banks by HTTP.
- `bank-a-service` uses the Seata XA MySQL driver to deduct money from `db_bank_a`.
- `bank-b-service` uses the Seata XA MySQL driver to add money to `db_bank_b`.

The global XID is sent in the HTTP header. Each bank service receives it through
the Gin transaction middleware, registers an XA branch with Seata, and lets Seata
coordinate the two-phase commit.

## Run

Start Seata and the two MySQL instances:

```bash
docker-compose -f xa/bank_transfer/docker-compose.yml up -d
```

Start the three services from the repository root:

```bash
go run ./xa/bank_transfer/bank-a-service
go run ./xa/bank_transfer/bank-b-service
go run ./xa/bank_transfer/transfer-service
```

Default ports:

- `transfer-service`: `18080`
- `bank-a-service`: `18081`
- `bank-b-service`: `18082`
- `mysql-bank-a`: `3307`
- `mysql-bank-b`: `3308`
- `seata-server`: `8091`

## Successful transfer

Check the initial balances:

```bash
curl http://127.0.0.1:18081/accounts/A-1001
curl http://127.0.0.1:18082/accounts/B-2001
```

Run a successful transfer:

```bash
curl -X POST http://127.0.0.1:18080/transfer/success
```

Expected result:

- `A-1001` in `db_bank_a` is debited by `100`.
- `B-2001` in `db_bank_b` is credited by `100`.
- Seata commits both XA branches.

## Gin and XA context

Bank services must enable `r.ContextWithFallback = true` on Gin 1.8.1+ so the
global XID injected by `TransactionMiddleware` is visible to `db.ExecContext`.

Database handlers should pass `c.Request.Context()` into SQL calls instead of
the `*gin.Context` value itself.

## Rollback transfer

`B-FROZEN` is a frozen account in `db_bank_b`. It rejects credits to simulate a
bank-b failure after bank-a has already executed the debit.

```bash
curl -X POST http://127.0.0.1:18080/transfer/fail
```

Expected result:

- `bank-a-service` first deducts `100` from `A-1001`.
- `bank-b-service` rejects the credit because `B-FROZEN` is frozen.
- `transfer-service` returns an error from the global transaction.
- Seata rolls back the bank-a XA branch, so `A-1001` keeps its original balance.

Verify balances:

```bash
curl http://127.0.0.1:18081/accounts/A-1001
curl http://127.0.0.1:18082/accounts/B-FROZEN
```

You can also send a custom request:

```bash
curl -X POST http://127.0.0.1:18080/transfer \
  -H 'Content-Type: application/json' \
  -d '{"from_account_no":"A-1001","to_account_no":"B-2001","amount":100}'
```

## Recover from a stuck XA state

If repeated rollback failures leave MySQL with prepared XA branches or row
locks, restart the three Go services and clean the databases:

```bash
docker exec mysql-bank-a mysql -uroot -p123456 -e "XA RECOVER;"
docker exec mysql-bank-b mysql -uroot -p123456 -e "XA RECOVER;"
```

For every row returned by `XA RECOVER`, run `XA ROLLBACK '<data>';` on that
database. Then verify no transaction is waiting:

```bash
docker exec mysql-bank-a mysql -uroot -p123456 -e "SELECT trx_id, trx_state FROM information_schema.innodb_trx;"
docker exec mysql-bank-b mysql -uroot -p123456 -e "SELECT trx_id, trx_state FROM information_schema.innodb_trx;"
```

## Environment variables

The sample defaults are ready for the compose file above. Override these when
running services against another environment:

- `SEATA_CONFIG`
- `BANK_A_URL`, `BANK_B_URL`
- `BANK_A_MYSQL_HOST`, `BANK_A_MYSQL_PORT`, `BANK_A_MYSQL_USERNAME`, `BANK_A_MYSQL_PASSWORD`, `BANK_A_MYSQL_DB`
- `BANK_B_MYSQL_HOST`, `BANK_B_MYSQL_PORT`, `BANK_B_MYSQL_USERNAME`, `BANK_B_MYSQL_PASSWORD`, `BANK_B_MYSQL_DB`
