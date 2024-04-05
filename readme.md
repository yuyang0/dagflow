# DAGFlow
====
![](https://github.com/yuyang0/dagflow/workflows/test/badge.svg)
![](https://github.com/yuyang0/dagflow/workflows/golangci-lint/badge.svg)
[![Codacy Badge](https://app.codacy.com/project/badge/Grade/e76c475f817a409d860934a64c603cb1)](https://app.codacy.com/gh/yuyang0/dagflow/dashboard?utm_source=gh&utm_medium=referral&utm_content=&utm_campaign=Badge_grade)

a DAG task engine based on asynq

## QuickStart
1. prepare asynq server mux and client
    ```golang
    srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr},
		asynq.Config{
			// Specify how many concurrent workers to use
			Concurrency: 10,
			// Optionally specify multiple queues with different priority.
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
			// See the godoc for other configuration options
		},
	)
    mux := asynq.NewServeMux()
    client := asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr})
    defer client.Close()
    ```
    please refer to asynq's doc to figure out more config options.
2. create dagflow service
    ```golang
    svc, err := service.New(&types.Config{
	    Store: types.StoreConfig{
		    Type: "redis",
		    Redis: types.RedisConfig{
		    	Addr:   "127.0.0.1:6379",
		    	Expire: 120,
		    },
	    },
    }, nil)
    if err != nil {
    	log.Fatal("failed to create service", err)
    }
    ```
3. create a flow object and register it to dagflow service
    ```golang
    f, err := svc.NewFlow("f1")
	if err != nil {
		log.Fatal("failed to create flow", err)
	}
	if err = f.Node("n1", incOp); err != nil {
		log.Fatal("failed to create node", err)
    }
    svc.RegisterFlows(mux, f)
    ```
4. start asynq server
    ```golang
    if err := srv.Run(mux); err != nil {
		log.Fatalf("could not run server: %v", err)
	}
    ```
5. submit dagflow tasks
    ```golang
    svc.Submit("f1", []byte(`1`))
    ```