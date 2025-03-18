package main

import (
	"encoding/json"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/hibiken/asynq"
	"github.com/yuyang0/dagflow/flow"
	"github.com/yuyang0/dagflow/service"
	"github.com/yuyang0/dagflow/types"
)

const redisAddr = "127.0.0.1:6379"
const flowName, nodeName = "f1", "n1"

func main() {
	if len(os.Args) != 2 {
		log.Fatal("usage: one_node [server|client]")
	}
	switch os.Args[1] {
	case "server":
		runServer()
	case "client":
		runClient()
	}
}

func runServer() {
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
	createSvcAndFlow(mux, flowName, nodeName)
	// ...register other handlers...

	if err := srv.Run(mux); err != nil {
		log.Fatalf("could not run server: %v", err)
	}
}

func runClient() {
	clientSVC, _ := createSvcAndFlow(nil, flowName, nodeName)
	sessID, err := clientSVC.Submit(flowName, []byte(`1`))
	if err != nil {
		log.Fatal("failed to submit task", err)
	}
	time.Sleep(15 * time.Second)
	resMap, err := clientSVC.GetResult(flowName, sessID)
	if err != nil {
		log.Fatal("failed to get result", err)
	}

	for k, v := range resMap {
		var i int
		json.Unmarshal(v.Resp, &i)
		log.Printf("+++++ %s, result: %d\n", k, i)
	}
}

func createSvcAndFlow(mux *asynq.ServeMux, flowName, nodeName string) (*service.Service, *flow.Flow) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	svc, err := service.New(&types.Config{
		Redis: types.RedisConfig{
			Addr:   redisAddr,
			Expire: 120,
		},
	}, logger)
	if err != nil {
		log.Fatal("failed to create service", err)
	}
	f, err := svc.NewFlow(flowName)
	if err != nil {
		log.Fatal("failed to create flow", err)
	}
	err = f.Node(nodeName, incOp)
	if err != nil {
		log.Fatal("failed to create node", err)
	}
	if err := svc.RegisterFlows(mux, f); err != nil {
		log.Fatalf("failed to register flow: %v", err)
	}
	return svc, f
}

func incOp(data []byte, option map[string][]string) ([]byte, error) {
	var i int
	if err := json.Unmarshal(data, &i); err != nil {
		return data, err
	}
	i++
	newData, err := json.Marshal(i)
	return newData, err
}
