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

## How to test samples for new PR

1. Modify the seata-go dependency version to v0.0.0-incompatible, and remove the version number if it exists in the original dependency path.

   ````
   //github.com/seata/seata-go v1.0.3
   github.com/seata/seata-go v0.0.0-incompatibl
   ````

2. Find the absolute or relative path to your local code.

   ````
   /Users/mac/Desktop/GO/seata-go
   ../seata-go
   ````

3. Add the replace module to the go.mod file and change it to the local code path.
   
   ````
   github.com/seata/seata-go =>  ../seata-go
   ````
   
4. Synchronization dependencies.

   ````shell
   go mod tidy
   ````

5. Run the sample test code.