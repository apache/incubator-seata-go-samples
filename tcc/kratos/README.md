# Kratos 接入 seata-go 例子

本案例中总共包含三个组件：
1. Server: 一个基于 kratos 的服务
2. Client: 一次分布式事务调用
3. TC: seata-server 事务协调器

## 部署TC
```shell
cd dockercompose
docker-compose -f docker-compose.yml up -d seata-server
```
## 启动 Server
```shell
cd tcc/kratos/server
make build && ./bin/kratos -conf configs/config.yaml
```
## Client 调用
```shell
cd tcc/kratos/client
go run main.go
```

