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

# AT E-commerce Sample

This sample demonstrates an e-commerce order flow in Seata Go AT mode.

The scenario contains three independent Go services:

1. `order-service` creates the order record and starts the global transaction
2. `inventory-service` deducts stock
3. `account-service` deducts the user balance

All three operations must succeed or fail together. If the account balance is insufficient, `account-service` rejects the request and Seata automatically rolls back the order creation and inventory deduction through `undo_log`.

## Directory Layout

- `order/`: `order-service`
- `inventory/`: `inventory-service`
- `account/`: `account-service`
- `sql/mysql_ecommerce.sql`: schema and seed data for the three MySQL databases
- `docker-compose.yml`: MySQL and Seata Server

## Start the Infrastructure

```bash
cd at/ecommerce
docker-compose up -d
```

Default ports:

- MySQL: `3306`
- Seata Server: `8091`
- order-service: `18080`
- inventory-service: `18081`
- account-service: `18082`

## Start the Three Services

From the repository root, open three terminals and run:

```bash
go run ./at/ecommerce/order
go run ./at/ecommerce/inventory
go run ./at/ecommerce/account
```

The services use `conf/seatago.yml`, so run them from the repository root as shown above.

## Run the Success Scenario

The account seed balance is `50`, so a `money` value below that limit commits successfully:

```bash
curl -X POST http://127.0.0.1:18080/createOrder \
  -H 'Content-Type: application/json' \
  -d '{"userId":"U100001","commodityCode":"C100001","count":2,"money":30}'
```

Expected result:

- a new row is inserted into `seata_ecommerce_order.order_tbl`
- `seata_ecommerce_inventory.inventory_tbl.stock` decreases from `100` to `98`
- `seata_ecommerce_account.account_tbl.balance` decreases from `50` to `20`

## Run the Rollback Scenario

This request exceeds the seed balance and makes `account-service` reject the deduction:

```bash
curl -X POST http://127.0.0.1:18080/createOrder \
  -H 'Content-Type: application/json' \
  -d '{"userId":"U100001","commodityCode":"C100001","count":2,"money":100}'
```

Expected result:

- `account-service` returns `balance not enough`
- the global transaction fails in `order-service`
- no new committed order row remains in `seata_ecommerce_order.order_tbl`
- inventory stock is rolled back to its previous value

## Verify in MySQL

```bash
mysql -h127.0.0.1 -P3306 -uroot -p123456 -e "SELECT * FROM seata_ecommerce_order.order_tbl;"
mysql -h127.0.0.1 -P3306 -uroot -p123456 -e "SELECT * FROM seata_ecommerce_inventory.inventory_tbl;"
mysql -h127.0.0.1 -P3306 -uroot -p123456 -e "SELECT * FROM seata_ecommerce_account.account_tbl;"
```
