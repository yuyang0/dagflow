package flow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/hibiken/asynq"
	"github.com/yuyang0/dagflow/types"
	"github.com/yuyang0/dagflow/utils"
	"github.com/yuyang0/dagflow/utils/idgen"
)

type ExecResult struct {
	ID   string `json:"id"`
	Resp []byte `json:"resp"`
	Err  string `json:"err"`
}

type Executor struct {
	// id format: "flow_name:node_name:random_id"
	ID   string `json:"id"`
	Body []byte `json:"body"`

	f *Flow `json:"-"`
	t *asynq.Task
}

func (e *Executor) Execute(ctx context.Context) error {
	f := e.f
	logger := f.logger
	flowName, nodeName, sessID, err := ParseExecID(e.ID)
	if err != nil {
		return err
	}
	dagNode, err := f.DAG.GetVertex(nodeName)
	if err != nil {
		return err
	}
	if flowName != f.Name {
		return errors.Newf("inconsist flow name: %s != %s", flowName, f.Name)
	}
	node := dagNode.(flowNode) // nolint
	resp, err := node.Execute(e.Body, nil)
	if err != nil { //nolint
		return utils.Invoke(func() (err3 error) {
			execOpts := node.ExecutionOptions()
			retried, _ := asynq.GetRetryCount(ctx)
			maxRetry, _ := asynq.GetMaxRetry(ctx)
			logger.Debug("exec failed", "retried", retried, "maxRetry", maxRetry, "err", err)
			defer func() {
				if retried >= maxRetry {
					eRes := &ExecResult{ID: e.ID, Resp: resp, Err: err3.Error()}
					if err := e.setExecResult(ctx, eRes); err != nil {
						logger.Error("failed to set result", "execID", e.ID, "err", err.Error())
					}
				}
			}()
			if execOpts.failureHandler != nil {
				if err1 := execOpts.failureHandler(e.Body, err); err1 != nil {
					return errors.CombineErrors(err, err1)
				}
			}
			if retried >= maxRetry {
				if execOpts.finalFailureHandler != nil {
					if err2 := execOpts.finalFailureHandler(e.Body, err); err2 != nil {
						return errors.CombineErrors(err, err2)
					}
				}
			}
			return err
		})
	}
	eRes := &ExecResult{ID: e.ID, Resp: resp, Err: ""}

	if err := e.setExecResult(ctx, eRes); err != nil {
		return errors.Wrapf(err, "failed to set result for %s", e.ID)
	}
	// submit children
	return e.submitChildren(node, sessID)
}

func (e *Executor) submitChildren(node flowNode, sessID string) error {
	f := e.f
	logger := f.logger
	childMap, err := f.DAG.GetChildren(node.Name())
	if err != nil {
		return err
	}
	for _, dagChild := range childMap {
		child := dagChild.(flowNode) // nolint
		logger.Debug("submit child", "node", node.Name(), "child", child.Name())
		if _, err := e.submitNode(child, sessID, nil); err != nil {
			if errors.Is(err, types.ErrExecHasDependency) {
				continue
			}
			return err
		}
	}
	return nil
}

func (e *Executor) submitRoots(body []byte) (sessID string, err error) {
	f := e.f
	sessID = idgen.NextSID()
	roots := f.DAG.GetRoots()
	for _, root := range roots {
		node := root.(flowNode) // nolint
		if _, err := e.submitNode(node, sessID, body); err != nil {
			return "", err
		}
	}
	return sessID, nil
}

func (e *Executor) submitNode(node flowNode, sessID string, body []byte) (*asynq.TaskInfo, error) { //nolint
	f := e.f
	cli := f.cli
	cfg := f.cfg
	var err error
	if body == nil {
		if body, err = e.getAggData(node, sessID); err != nil {
			return nil, err
		}
	}
	newExec := &Executor{
		ID:   newExecID(f.Name, node.Name(), sessID),
		Body: body,
	}
	bs, _ := json.Marshal(newExec)
	asynT := asynq.NewTask(f.Name, bs)
	opts := []asynq.Option{
		asynq.TaskID(newExec.ID),
		asynq.MaxRetry(cfg.RetryCount),
		asynq.Retention(cfg.Timeout),
		asynq.Timeout(cfg.Timeout),
	}
	return cli.Enqueue(asynT, opts...)
}

func (e Executor) getAggData(node flowNode, sessID string) ([]byte, error) {
	f := e.f
	logger := f.logger
	parents, err := f.DAG.GetParents(node.Name())
	if err != nil {
		return nil, err
	}
	aggData := map[string][]byte{}
	for _, dagParent := range parents {
		parent := dagParent.(flowNode) // nolint
		execID := newExecID(f.Name, parent.Name(), sessID)
		eRes, err := e.getExecResult(execID)
		logger.Debug("parent result", "execID", execID, "eRes", eRes, "err", err)
		if err != nil {
			return nil, err
		}
		if eRes == nil {
			return nil, errors.Wrapf(types.ErrExecHasDependency, "dependency: %s", execID)
		}
		aggData[parent.Name()] = eRes.Resp
	}
	var body []byte
	execOpts := node.ExecutionOptions()
	if execOpts.aggregator != nil {
		body, err = execOpts.aggregator(aggData)
	} else {
		body, err = randomAgg(aggData)
	}
	return body, errors.Wrapf(err, "failed to aggregate parent node's response")
}

func (e *Executor) getLeavesResult(sessID string) (map[string]*ExecResult, error) {
	f := e.f
	resMap := map[string]*ExecResult{}
	leaves := f.DAG.GetLeaves()
	for _, leaf := range leaves {
		node := leaf.(flowNode) // nolint
		execID := newExecID(f.Name, node.Name(), sessID)
		eRes, err := e.getExecResult(execID)
		if err != nil {
			return nil, err
		}
		resMap[node.Name()] = eRes
	}
	return resMap, nil
}

func (e *Executor) setExecResult(ctx context.Context, eRes *ExecResult) error {
	cfg := e.f.cfg
	bs, err := json.Marshal(eRes)
	if err != nil {
		return err
	}
	if cfg.UseCustomStore {
		err = e.f.stor.Set(ctx, eRes.ID, bs)
	} else {
		_, err = e.t.ResultWriter().Write(bs)
	}
	return err
}

func (e *Executor) getExecResult(execID string) (eRes *ExecResult, err error) {
	var bs []byte
	f := e.f
	cfg := e.f.cfg
	if cfg.UseCustomStore {
		bs, err = f.stor.Get(context.TODO(), execID)
	} else {
		ti, err := f.insp.GetTaskInfo("default", execID)
		if err != nil {
			return nil, err
		}
		bs = ti.Result
	}
	if err != nil {
		return nil, err
	}
	if bs == nil {
		return nil, nil
	}
	eRes = &ExecResult{}
	if err := json.Unmarshal(bs, eRes); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal json for ExecResult")
	}
	return eRes, nil
}

func newExecID(flowName, nodeName, randomID string) string {
	return fmt.Sprintf("%s%s%s%s%s", flowName, nameSep, nodeName, nameSep, randomID)
}

func ParseExecID(execID string) (flowName string, nodeName string, sessID string, err error) {
	parts := strings.Split(execID, nameSep)
	if len(parts) != 3 {
		err = errors.Newf("failed to parse execution id")
		return
	}
	flowName, nodeName, sessID = parts[0], parts[1], parts[2]
	return
}
