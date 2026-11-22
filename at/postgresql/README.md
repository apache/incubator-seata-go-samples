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

# AT mode with PostgreSQL

This sample uses the `seata-at-postgres` driver from seata-go v2 with plain
`database/sql`. It runs two global transactions against one PostgreSQL
database:

- the first one takes 100 from `account_tbl` and inserts a row into
  `order_tbl`, then commits;
- the second one does the same and then returns an error, so Seata rolls it
  back using the undo log and the data goes back to what it was.

## Run

Start Seata and PostgreSQL:

```bash
docker-compose -f at/postgresql/docker-compose.yml up -d
```

`sql/init.sql` creates the `undo_log` table and the two business tables. AT
mode needs the `undo_log` table in the same database as your business tables.

Give PostgreSQL a few seconds to start before running the sample. The compose
file uses ports 5432, 8091 and 7091, so stop a local PostgreSQL or the
seata-server from `dockercompose/` first if they are running.

Run the sample from the repository root:

```bash
go run ./at/postgresql
```

Among the Seata log lines you will see (the numbers change on every run):

```text
before: balance=1000 orders=0
global transaction started, xid=...
after commit: balance=900 orders=1
undo log cleaned up after commit
global transaction started, xid=...
after rollback: balance=900 orders=1
rollback restored the data
```

## Using the driver

Open the database with the Seata PostgreSQL AT driver and a pgx DSN:

```go
import (
	"database/sql"

	seataSQL "seata.apache.org/seata-go/v2/pkg/datasource/sql"
)

db, err := sql.Open(seataSQL.SeataATPostgresDriver,
	"postgres://seata:seata@127.0.0.1:5432/seata_at?sslmode=disable")
```

Use PostgreSQL placeholders (`$1`, `$2`, ...) in your SQL, and pass the
context of the global transaction to `ExecContext` / `BeginTx`.

## Environment variables

- `POSTGRES_DSN`: PostgreSQL DSN, default
  `postgres://seata:seata@127.0.0.1:5432/seata_at?sslmode=disable`
- `SEATA_CONFIG`: path to `seatago.yml`, default `conf/seatago.yml`
