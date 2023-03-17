## Starting the app

* clone the `go_meetup_api` repo as a sibling of this repo:
  * ```cd .. && git clone https://github.com/minauteur/go_meetup_api```

* Then, make sure you have `curl` and `grpcurl` installed; for MacOS users:
  * ```brew install curl grpcurl```

* Next, install `go` version `1.19` or newer:
  * If you already have and use an earlier version of go, run both the following to install a subversion usable as `go1.19.5`:
    * ```go install golang.org/dl/go1.19.5@latest```
    * ```go1.19.5 download```

* Finally, from the `go_meetup_server` root directory, run the following:
  * ```go1.19.5 run cmd/app/main.go```


## Examples

**NOTE:** These example commands need to be run from `go_meetup_api` root for the import path and proto flags to be correct

* To Demonstrate Multiplexing of Http/REST and gRPC/connect requests on a single port:
  * *curl*
    ```
    curl -X POST --data '{}' --header 'Content-Type: application/json' http://0.0.0.0:8080/api.greeting.v1.GreetingAPI/Greet
    ```
  * *grpcurl*
    ```
    grpcurl -plaintext -import-path ./proto -proto ./proto/api/greeting/v1/service.proto -d '{"name":"Me", "entity_type": "ENTITY_TYPE_HUMAN"}' 0.0.0.0:8080 api.greeting.v1.GreetingAPI/Greet
    ```
 * To Demonstrate Graceful Shutdown:
 
    **NOTE:** The timeout is set to 10 seconds in `main.go`--in order to test graceful shutdown, you must quickly interrupt the server (e.g. press `Ctrl+c` in the shell where the server is running) _after_ sending one of the following requests:
    
   * *example interrupt with wait time _within_ the timeout period:*
     ``` 
     grpcurl -plaintext -import-path ./proto -proto ./proto/api/wait/v1/service.proto -d '{"wait_time": 5}' 0.0.0.0:8080 api.wait.v1.WaitAPI/Wait
     ```
   * *example interrupt with wait time **longer** than the timeout (e.g. timeout expires):*
     ```
     grpcurl -plaintext -import-path ./proto -proto ./proto/api/wait/v1/service.proto -d '{"wait_time": 20}' 0.0.0.0:8080 api.wait.v1.WaitAPI/Wait
     ```
     
  * To Demonstrate Interceptor and Fieldmask Behavior:
    * *with "admin" authorization (all fields returned):*
      ```
      grpcurl -plaintext -import-path ./proto -proto ./proto/api/record/v1/service.proto -H 'Authorization: valid' 0.0.0.0:8080 api.record.v1.RecordAPI/Get
      ```
    * *with "non-admin" authorization (public fields only returned):*
      ```
      grpcurl -plaintext -import-path ./proto -proto ./proto/api/record/v1/service.proto -H 'Authorization: INvalid' 0.0.0.0:8080 api.record.v1.RecordAPI/Get
      ```
