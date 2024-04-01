package flow

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/cockroachdb/errors"
	"github.com/heimdalr/dag"
	"github.com/hibiken/asynq"
	"github.com/yuyang0/dagflow/store"
)

type NodeFunc func([]byte, map[string][]string) ([]byte, error)

type Flow struct {
	Name   string
	DAG    *dag.DAG
	stor   store.Store
	cli    *asynq.Client
	logger *slog.Logger
}

func New(name string, stor store.Store, cli *asynq.Client, logger *slog.Logger) *Flow {
	return &Flow{
		Name:   name,
		DAG:    dag.NewDAG(),
		stor:   stor,
		cli:    cli,
		logger: logger,
	}
}

type FlowNode struct {
	name     string
	fn       NodeFunc
	execOpts *ExecutionOptions
}

func (f *Flow) Node(name string, fn NodeFunc, opts ...Option) error {
	execOpts := &ExecutionOptions{}
	for _, opt := range opts {
		opt(execOpts)
	}
	return f.DAG.AddVertexByID(name, &FlowNode{
		name:     name,
		fn:       fn,
		execOpts: execOpts,
	})
}

func (f *Flow) Edge(src, dst string) error {
	return f.DAG.AddEdge(src, dst)
}

func (f *Flow) Register(mux *asynq.ServeMux) {
	mux.HandleFunc(f.Name, f.asynqHandler())
}

func (f *Flow) Submit(body []byte) (string, error) {
	e := &Executor{
		f:   f,
		cli: f.cli,
	}
	return e.submitRoots(body)
}

func (f *Flow) GetResult(sessID string) (map[string]*ExecResult, error) {
	e := &Executor{
		f:   f,
		cli: f.cli,
	}
	return e.getLeavesResult(sessID)
}

func (f *Flow) asynqHandler() asynq.HandlerFunc {
	return func(ctx context.Context, t *asynq.Task) error {
		eInfo := &ExecInfo{}
		if err := json.Unmarshal(t.Payload(), eInfo); err != nil {
			return errors.Wrapf(err, "filed to unmarshal payload")
		}
		e := &Executor{
			f:   f,
			cli: f.cli,
		}
		if err := e.Execute(eInfo); err != nil {
			f.logger.Error("failed to run task", "execID", eInfo.ID, "err", err)
			return err
		}
		return nil
	}
}
