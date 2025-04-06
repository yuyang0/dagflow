# DAGFlow

![lint](https://github.com/yuyang0/dagflow/workflows/test/badge.svg)
![UT](https://github.com/yuyang0/dagflow/workflows/golangci-lint/badge.svg)
[![Codacy Badge](https://app.codacy.com/project/badge/Grade/e76c475f817a409d860934a64c603cb1)](https://app.codacy.com/gh/yuyang0/dagflow/dashboard?utm_source=gh&utm_medium=referral&utm_content=&utm_campaign=Badge_grade)
[![GoDoc](https://godoc.org/github.com/yuyang0/dagflow?status.svg)](https://godoc.org/github.com/yuyang0/dagflow)
[![License: MIT](https://img.shields.io/badge/license-MIT-green.svg)](https://opensource.org/licenses/MIT)

a DAG task engine based on [asynq](https://github.com/hibiken/asynq)

## QuickStart
The examples directory contains serveral examples, it's a good place to figure out how to use this library.
### server side
1. prepare asynq server mux
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
    ```
    please refer to asynq's doc to figure out more config options.
2. create dagflow service
    ```golang
    svc, err := service.New(&types.Config{
		Redis: types.RedisConfig{
			Addr:   "127.0.0.1:6379",
			Expire: 120,
		},
	    Store: types.StoreConfig{
		    Type: "redis",
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
    // for complex dag, you can use RegisterFlowsWithDefinitor
    svc.RegisterFlows(mux, f)
    ```
4. start asynq server
    ```golang
    if err := srv.Run(mux); err != nil {
		log.Fatalf("could not run server: %v", err)
	}
    ```
### client side
1. create dagflow service, same as step 2 in server side
    ```golang
    svc, err := service.New(&types.Config{
		Redis: types.RedisConfig{
			Addr:   "127.0.0.1:6379",
			Expire: 120,
		},
	    Store: types.StoreConfig{
		    Type: "redis",
	    },
    }, nil)
    if err != nil {
    	log.Fatal("failed to create service", err)
    }
    ```
2. create a flow object and register it to dagflow service, same as step 3 in server side except `mux` should be set to `nil`
    ```golang
    f, err := svc.NewFlow("f1")
	if err != nil {
		log.Fatal("failed to create flow", err)
	}
	if err = f.Node("n1", incOp); err != nil {
		log.Fatal("failed to create node", err)
    }
    // for complex dag, you can use RegisterFlowsWithDefinitor
    svc.RegisterFlows(nil, f)
    ```
3. submit dagflow tasks
    ```golang
    svc.Submit("f1", []byte(`1`))
    ```

## DAG
### single node DAG

```golang
f, err := svc.NewFlow("f1")
if err != nil {
    log.Fatal("failed to create flow", err)
}
if err = f.Node("n1", incOp); err != nil {
    log.Fatal("failed to create node", err)
}
```

### complex DAG
```golang

func prepareFlow(f *flow.Flow) error {
	if err := f.Node("l1n1", incOp); err != nil {
		return err
	}
	if err := f.Node("l2n1", incOp); err != nil {
		return err
	}
	if err := f.Node("l2n2", decOp); err != nil {
		return err
	}
	if err := f.Node("l3n1", mulOp, flow.WithAggregator(func (dataMap map[string][]byte) ([]byte, error) {
        l2n1Result := dataMap["l2n1"]
        l2n2Result := dataMap["l2n2"]
        // do anything you want to construct input data for node l3n1
    })); err != nil {
		return err
	}
	if err := f.Edge("l1n1", "l2n1"); err != nil {
		return err
	}
	if err := f.Edge("l1n1", "l2n2"); err != nil {
		return err
	}
	if err := f.Edge("l2n1", "l3n1"); err != nil {
		return err
	}
	if err := f.Edge("l2n2", "l3n1"); err != nil {
		return err
	}
	return nil
}
```

### SwitchNode
SwitchNode is a special type node which works like switch case statment in golang

```golang

f, err := svc.NewFlow("f1")
if err != nil {
    log.Fatal("failed to create flow", err)
}
if err = f.SwitchNode("n1", func(data []byte) string {
    return "+"
}, map[string]flow.NodeFunc{
    "+": incOp,
    "-": decOp,
}); err != nil {
    log.Fatal("failed to create node", err)
}
```
