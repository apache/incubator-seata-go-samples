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

CREATE DATABASE IF NOT EXISTS seata_ride DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE seata_ride;

-- ride orders
CREATE TABLE IF NOT EXISTS ride_orders (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    passenger_id VARCHAR(64) NOT NULL,
    status TINYINT NOT NULL DEFAULT 0 COMMENT '0=pending, 1=confirmed, 2=cancelled',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- drivers
CREATE TABLE IF NOT EXISTS drivers (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(64) NOT NULL,
    status TINYINT NOT NULL DEFAULT 0 COMMENT '0=available, 1=reserved, 2=busy',
    reserved_order_id BIGINT DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO drivers (name, status) VALUES
    ('Driver-A', 0),
    ('Driver-B', 0),
    ('Driver-C', 0);

-- price locks
CREATE TABLE IF NOT EXISTS price_locks (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    order_id BIGINT NOT NULL,
    amount INT NOT NULL COMMENT 'fare in cents',
    status TINYINT NOT NULL DEFAULT 0 COMMENT '0=locked, 1=confirmed, 2=released'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- coupons
CREATE TABLE IF NOT EXISTS coupons (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id VARCHAR(64) NOT NULL,
    discount INT NOT NULL COMMENT 'discount in cents',
    status TINYINT NOT NULL DEFAULT 0 COMMENT '0=available, 1=frozen, 2=used'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO coupons (user_id, discount, status) VALUES
    ('passenger-001', 500, 0),
    ('passenger-001', 300, 0);

-- vehicle capacity
CREATE TABLE IF NOT EXISTS vehicle_capacity (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    vehicle_type VARCHAR(32) NOT NULL,
    total_slots INT NOT NULL,
    reserved_slots INT NOT NULL DEFAULT 0
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO vehicle_capacity (vehicle_type, total_slots, reserved_slots) VALUES
    ('standard', 10, 0);
