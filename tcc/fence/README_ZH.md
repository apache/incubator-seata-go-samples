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

## 用例介绍
此用例介绍如何在tcc本地模式下使用防悬挂功能

## 使用步骤

- 在您的数据库中使用``./sample/tcc/fence/script/mysql.sql``脚本创建防悬挂所需的日志记录表，如果您使用的是其他数据库则运行对应数据库的脚本文件。
- 在``./sample/tcc/fence/service/service.go``中修改数据库驱动名为对应数据库类型并引入相关驱动包，mysql无需修改。此外需要注意用户名和密码是否正确。
- 启动``seata tc server``
- 使用以下命令运行用例``go run ./sample/tcc/fence/cmd/main.go``