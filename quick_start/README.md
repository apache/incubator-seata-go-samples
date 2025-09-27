# Seata Go 快速入门指南

## 项目概述

## 环境准备

### 1. 前置要求
- Go 1.18 或更高版本
- Docker 环境
- MySQL 数据库（通过 Docker 启动）
- Seata Server

### 2. 启动 Seata Server

首先，使用项目提供的 Docker Compose 文件启动 Seata Server 服务：

```shell
# 克隆代码库
https://github.com/apache/incubator-seata-go-samples.git
cd incubator-seata-go-samples

# 启动 Seata Server
docker-compose -f dockercompose/docker-compose.yml up -d seata-server
```

## 快速开始：构建您的第一个 Seata Go 应用

本指南将使用 quick_start 模块来展示基本的分布式事务示例。

### 1. 项目结构

quick_start 模块的基本结构如下：

```
quick_start/
├── cmd/            # 应用入口
│   ├── main.go     # 主程序入口
│   └── db.go       # 数据库配置与初始化
├── handler/        # HTTP 请求处理器
│   ├── order.go    # 订单相关 API 处理器
│   └── vo.go       # 视图对象定义
├── model/          # 数据模型
│   └── order.go    # 订单模型
└── service/        # 业务逻辑层
    └── order.go    # 订单服务
```

### 2. 核心组件说明

#### 数据模型 (model/order.go)

定义了订单实体，映射数据库表结构：

```go
package model

type Order struct {
    ID            int64  `gorm:"primaryKey;autoIncrement;column:id"`
    UserID        string `gorm:"column:user_id;type:varchar(255)"`
    CommodityCode string `gorm:"column:commodity_code;type:varchar(255)"`
    Count         int64  `gorm:"column:count;default:0"`
    Money         int64  `gorm:"column:money;default:0"`
    IsDeleted     string `gorm:"column:is_deleted"`
    Utime         int64  `gorm:"column:utime"`
    Ctime         int64  `gorm:"column:ctime"`
}

func (Order) TableName() string {
    return "orders"
}
```

#### 数据库配置 (cmd/db.go)

配置 Seata 事务驱动和数据库连接：

```go
package cmd

import (
    "database/sql"
    "gorm.io/driver/mysql"
    "gorm.io/gorm"
    sql2 "seata.apache.org/seata-go/pkg/datasource/sql"
    "sync"
)

var (
    gormDB *gorm.DB
    dbOnce sync.Once
)

func DB() {
    dbOnce.Do(func() {
        // 使用 Seata XA 驱动
        sqlDB, err := sql.Open(
            sql2.SeataXAMySQLDriver,
            "root:12345678@tcp(127.0.0.1:3306)/seata_client?multiStatements=true&interpolateParams=true")
        if err != nil {
            panic("init service error")
        }

        gormDB, err = gorm.Open(mysql.New(mysql.Config{
            Conn: sqlDB,
        }), &gorm.Config{})

        if err != nil {
            panic("failed to create the db")
        }
    })
}
```

#### 服务层 (service/order.go)

实现业务逻辑，包含分布式事务操作：

```go
package service

import (
    "context"
    "gorm.io/gorm"
    "seata.apache.org/seata-go-samples/quick_start/model"
    "time"
)

type OrderService struct {
    db *gorm.DB
}

func NewOrderDao(db *gorm.DB) *OrderService {
    return &OrderService{db: db}
}

func (o *OrderService) Create(ctx context.Context, order model.Order) (int64, error) {
    now := time.Now().Unix()
    order.Ctime = now
    order.Utime = now
    err := o.db.WithContext(ctx).Create(order).Error
    if err != nil {
        return 0, err
    }
    return order.ID, nil
}

func (o *OrderService) Delete(ctx context.Context, id int64) error {
    return o.db.WithContext(ctx).
        Model(&model.Order{}).
        Where("id = ?", id).
        Update("is_deleted", "true").
        Error
}
```

#### 控制器 (handler/order.go)

处理 HTTP 请求，调用服务层接口：

```go
package handler

import (
    "github.com/gin-gonic/gin"
    "net/http"
    "seata.apache.org/seata-go-samples/quick_start/model"
    "seata.apache.org/seata-go-samples/quick_start/service"
)

type OrderHandler struct {
    svc *service.OrderService
}

func NewOrderHandler(svc *service.OrderService) *OrderHandler {
    return &OrderHandler{
        svc: svc,
    }
}

func (o *OrderHandler) Route(engine *gin.Engine) {
    group := engine.Group("/order")
    group.POST("/create", o.Create)
    group.DELETE("/delete/:id", o.Delete)
}

// 其他方法略...
```

### 3. 配置分布式事务

使用 Seata Go 的关键在于正确配置和使用分布式事务。以下是基本配置步骤：

1. **配置 Seata 客户端**

   通常在项目启动时初始化 Seata 客户端配置：

   ```go
   import (
       "seata.apache.org/seata-go/pkg/client"
   )
   
   func initConfig() {
       // 加载 Seata 配置文件
       client.InitPath("../../conf/seatago.yml")
   }
   ```

2. **使用分布式事务**

   在需要事务支持的业务方法上，使用 `tm.WithGlobalTx` 包装：

   ```go
   import (
       "context"
       "time"
       "seata.apache.org/seata-go/pkg/tm"
   )
   
   // 在分布式事务中执行操作
   err := tm.WithGlobalTx(context.Background(), &tm.GtxConfig{
       Name:    "CreateOrderTx",
       Timeout: time.Second * 30,
   }, func(ctx context.Context) error {
       // 事务中的业务逻辑
       _, err := orderService.Create(ctx, order)
       return err
   })
   ```

## 自定义配置

### MySQL 连接配置

您可以通过系统环境变量自定义 MySQL 连接配置：

1. MYSQL_HOST - MySQL 主机地址
2. MYSQL_PORT - MySQL 端口
3. MYSQL_USERNAME - MySQL 用户名
4. MYSQL_PASSWORD - MySQL 密码
5. MYSQL_DB - 数据库名称

## 测试示例

### 运行 Quick Start 示例

1. 确保 Seata Server 已经启动
2. 进入 quick_start 目录
3. 运行示例代码：

```shell
cd quick_start
# 根据实际入口文件调整命令
```

### 测试 API

使用 curl 或其他工具测试接口：

```shell
# 创建订单
curl -X POST http://localhost:8080/order/create -H "Content-Type: application/json" -d '{"user_id":"1001","money":100}'

# 删除订单
curl -X DELETE http://localhost:8080/order/delete/1
```

## 开发与调试技巧

### 使用 Go Workspace 测试 PR

如果您想测试尚未合并到主分支的 PR，可以使用 Go Workspace：

1. 确保 Go 版本为 1.18 或更高
2. 克隆 seata-go 和 seata-go-samples 项目到同一目录
3. 在父目录初始化 workspace：

```shell
go work init
go work use ./seata-go
go work use ./seata-go-samples
```

4. 现在您可以直接运行示例代码来测试本地的 seata-go 代码

## 总结

通过本快速入门指南，您已经了解了如何在 Go 项目中集成和使用 Seata 分布式事务框架。Seata Go 提供了简单而强大的 API，帮助您在分布式环境中确保数据一致性。

更多高级用法和详细配置，请参考项目中的其他示例目录，如 at/basic、tcc、xa 等，它们展示了 Seata 支持的不同事务模式和集成场景。
