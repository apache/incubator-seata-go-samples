# seata-go-samples

## How to run samples

1. Start the seata-server service with the docker file under the /dockercomposer folder

   ~~~shell
   git clone https://github.com/seata/seata-go-samples.git && cd dockercompose
   docker-compose -f docker-compose.yml up -d seata-server
   ~~~

2. Start and run a sample, for example let's run a basic distributed transaction sample of AT

   ~~~shell
   cd at/basic
   go run main.go
   ~~~
