package flow

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/heimdalr/dag"
	"github.com/hibiken/asynq"
	"github.com/yuyang0/dagflow/store"
	"github.com/yuyang0/dagflow/types"
)

const (
	nameSep = ":"
)

type NodeFunc func([]byte, map[string][]string) ([]byte, error)
type SwitchCondFunc func([]byte) string
type Definitor func(ctx context.Context, f *Flow) error

type Flow struct {
	Name   string
	DAG    *dag.DAG
	stor   store.Store
	cli    *asynq.Client
	logger *slog.Logger
	cfg    *types.Config
	insp   *asynq.Inspector
}

func New(
	name string, stor store.Store, cli *asynq.Client,
	logger *slog.Logger, cfg *types.Config, insp *asynq.Inspector,
) (*Flow, error) {
	if strings.Contains(name, nameSep) {
		return nil, errors.Newf("flow name can't contain %s", nameSep)
	}
	return &Flow{
		Name:   name,
		DAG:    dag.NewDAG(),
		stor:   stor,
		cli:    cli,
		logger: logger,
		cfg:    cfg,
		insp:   insp,
	}, nil
}

type flowNode struct {
	name     string
	fn       NodeFunc
	execOpts *ExecutionOptions

	// for switch node
	condFn SwitchCondFunc
	cases  map[string]NodeFunc
}

func (f *Flow) Node(name string, fn NodeFunc, opts ...Option) error {
	if strings.Contains(name, nameSep) {
		return errors.Newf("dag node name can't contain %s", nameSep)
	}
	execOpts := &ExecutionOptions{}
	for _, opt := range opts {
		opt(execOpts)
	}
	return f.DAG.AddVertexByID(name, &flowNode{
		name:     name,
		fn:       fn,
		execOpts: execOpts,
	})
}

func (f *Flow) SwitchNode(
	name string, condFn SwitchCondFunc,
	cases map[string]NodeFunc, opts ...Option,
) error {
	if strings.Contains(name, nameSep) {
		return errors.Newf("dag node name can't contain %s", nameSep)
	}
	execOpts := &ExecutionOptions{}
	for _, opt := range opts {
		opt(execOpts)
	}
	return f.DAG.AddVertexByID(name, &flowNode{
		name:     name,
		execOpts: execOpts,
		condFn:   condFn,
		cases:    cases,
	})
}

func (f *Flow) Edge(src, dst string) error {
	if strings.Contains(src, nameSep) {
		return errors.Newf("dag src node name can't contain %s", nameSep)
	}
	if strings.Contains(dst, nameSep) {
		return errors.Newf("dag dst node name can't contain %s", nameSep)
	}
	return f.DAG.AddEdge(src, dst)
}

func (f *Flow) Register(mux *asynq.ServeMux) {
	if mux == nil {
		return
	}
	mux.HandleFunc(f.Name, f.asynqHandler())
}

func (f *Flow) Submit(body []byte) (string, error) {
	e := &Executor{
		f: f,
	}
	return e.submitRoots(body)
}

func (f *Flow) GetResult(sessID string) (map[string]*ExecResult, error) {
	e := &Executor{
		f: f,
	}
	return e.getLeavesResult(sessID)
}

func (f *Flow) asynqHandler() asynq.HandlerFunc {
	return func(ctx context.Context, t *asynq.Task) error {
		e := &Executor{
			f: f,
			t: t,
		}
		if err := json.Unmarshal(t.Payload(), e); err != nil {
			return errors.Wrapf(err, "filed to unmarshal payload")
		}
		if err := e.Execute(ctx); err != nil {
			f.logger.Error("failed to run task", "execID", e.ID, "err", err)
			return err
		}
		return nil
	}
}
