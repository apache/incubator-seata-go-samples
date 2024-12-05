# seata-go-samples

## How to run samples

1. Start the seata-server service with the docker file under the /dockercomposer folder

   ```shell
   git clone https://seata.apache.org/seata-go-samples.git
   ```

   ```shell
   cd dockercompose
   ```

   ```shell
   docker-compose -f docker-compose.yml up -d seata-server
   ```

2. Start and run a sample, for example let's run a basic distributed transaction sample of AT

   ```shell
   cd at/basic
   ```

   ```shell
   go run .
   ```

### Customize mysql connection configurations

The default mysql connection configuration is suitable with dockercompose/docker-compose.yml.

You can also customize it by system environment.

System Env Supported:

1. MYSQL_HOST
2. MYSQL_PORT
3. MYSQL_USERNAME
4. MYSQL_PASSWORD
5. MYSQL_DB

## How to use go mod replace to test samples for new PR

1. Modify the seata-go dependency version to v0.0.0-incompatible, and remove the version number if it exists in the
   original dependency path.

   ```
   //seata.apache.org/seata-go v1.2.0
   seata.apache.org/seata-go v0.0.0-incompatibl
   ```

2. Find the absolute or relative path to your local code.

   ```
   /Users/mac/Desktop/GO/seata-go
   ../seata-go
   ```

3. Add the replace module to the go.mod file and change it to the local code path.

   ```
   seata.apache.org/seata-go =>  ../seata-go
   ```

4. Synchronization dependencies.

   ```shell
   go mod tidy
   ```

5. Run the sample test code.

## How to use go workspace to test samples for new PR (Recommended)

You can use your local project forked to test new pr that haven't merged into seata.apache.org/seata-go.

1. Make sure your go version is 1.18 or later.

2. Fork and download two projects (seata-go and seata-go-samples), and put them into a same local directory.

For examples:

```text
/Users/xxx/code/github.com/yourid/seata-go
/Users/xxx/code/github.com/yourid/seata-go-samples
```

3. Execute go work init command in projects root directory.

```shell
cd /Users/xxx/code/github.com/yourid
```

```shell
go work init
```

You will find a go.work file in /Users/xxx/code/github.com/yourid.

4. Add seata-go and seata-go-samples into the same go workspace.

```shell
go work use ./seata-go
```

```shell
go work use ./seata-go-samples
```

Now, the content of go.work file is as follows.

```text
go 1.19

use (
        ./seata-go
        ./seata-go-samples
)
```

5. Run the sample test code in seata-go-samples.
