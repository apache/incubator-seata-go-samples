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

-- undo_log table for AT mode, taken from Seata's script/client/at/db/postgresql.sql
CREATE TABLE IF NOT EXISTS undo_log
(
    id            SERIAL       NOT NULL,
    branch_id     BIGINT       NOT NULL,
    xid           VARCHAR(128) NOT NULL,
    context       VARCHAR(128) NOT NULL,
    rollback_info BYTEA        NOT NULL,
    log_status    INT          NOT NULL,
    log_created   TIMESTAMP(0) NOT NULL,
    log_modified  TIMESTAMP(0) NOT NULL,
    CONSTRAINT pk_undo_log PRIMARY KEY (id),
    CONSTRAINT ux_undo_log UNIQUE (xid, branch_id)
);
CREATE INDEX IF NOT EXISTS ix_log_created ON undo_log (log_created);

CREATE TABLE IF NOT EXISTS account_tbl
(
    id      BIGINT      NOT NULL,
    user_id VARCHAR(64) NOT NULL,
    balance BIGINT      NOT NULL DEFAULT 0,
    CONSTRAINT pk_account_tbl PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS order_tbl
(
    id             BIGSERIAL   NOT NULL,
    user_id        VARCHAR(64) NOT NULL,
    commodity_code VARCHAR(64) NOT NULL,
    count          INT         NOT NULL DEFAULT 0,
    money          BIGINT      NOT NULL DEFAULT 0,
    CONSTRAINT pk_order_tbl PRIMARY KEY (id)
);

INSERT INTO account_tbl (id, user_id, balance)
VALUES (1, 'U1001', 1000)
ON CONFLICT (id) DO NOTHING;
