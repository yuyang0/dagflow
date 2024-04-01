package flow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/hibiken/asynq"
	"github.com/yuyang0/dagflow/types"
	"github.com/yuyang0/dagflow/utils/idgen"
)

type ExecInfo struct {
	// id format: "flow_name:node_name:random_id"
	ID   string `json:"id"`
	Body []byte `json:"body"`
}

type ExecResult struct {
	ID   string `json:"id"`
	Resp []byte `json:"resp"`
	Err  string `json:"err"`
}

type Executor struct {
	f   *Flow
	cli *asynq.Client
}

func (e *Executor) Execute(eInfo *ExecInfo) error {
	f := e.f
	logger := f.logger
	flowName, nodeName, sessID, err := parseExecID(eInfo.ID)
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
	node := dagNode.(*FlowNode) // nolint
	resp, err := node.fn(eInfo.Body, nil)
	if err != nil {
		return err
	}
	if err := e.setExecResult(&ExecResult{ID: eInfo.ID, Resp: resp, Err: ""}); err != nil {
		return errors.Wrapf(err, "failed to set result for %s", eInfo.ID)
	}
	// submit children
	logger.Debug("submit children", "node", nodeName)
	return e.submitChildren(node, sessID)
}

func (e *Executor) submitChildren(node *FlowNode, sessID string) error {
	f := e.f
	logger := f.logger
	childMap, err := f.DAG.GetChildren(node.name)
	if err != nil {
		return err
	}
	for _, dagChild := range childMap {
		child := dagChild.(*FlowNode) // nolint
		logger.Debug("submit node", "node", child.name)
		if _, err := e.submitNode(child, sessID); err != nil {
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
		node := root.(*FlowNode) // nolint
		eInfo := &ExecInfo{
			ID:   newExecID(f.Name, node.name, sessID),
			Body: body,
		}
		bs, _ := json.Marshal(eInfo)
		asynT := asynq.NewTask(f.Name, bs, asynq.TaskID(eInfo.ID))
		if _, err := e.cli.Enqueue(asynT, nil); err != nil {
			return "", err
		}
	}
	return sessID, nil
}

func (e *Executor) submitNode(node *FlowNode, sessID string) (*asynq.TaskInfo, error) {
	f := e.f
	logger := f.logger
	parents, err := f.DAG.GetParents(node.name)
	if err != nil {
		return nil, err
	}
	aggData := map[string][]byte{}
	for _, dagParent := range parents {
		parent := dagParent.(*FlowNode) // nolint
		execID := newExecID(f.Name, parent.name, sessID)
		eRes, err := e.getExecResult(execID)
		logger.Debug("parent result", "execID", execID, "eRes", eRes, "err", err)
		if err != nil {
			return nil, err
		}
		if eRes == nil {
			return nil, errors.Wrapf(types.ErrExecHasDependency, "dependency: %s", execID)
		}
		aggData[parent.name] = eRes.Resp
	}
	var body []byte
	if node.execOpts.aggregator != nil {
		body, err = node.execOpts.aggregator(aggData)
	} else {
		body, err = randomAgg(aggData)
	}
	if err != nil {
		return nil, err
	}
	eInfo := &ExecInfo{
		ID:   newExecID(f.Name, node.name, sessID),
		Body: body,
	}
	bs, _ := json.Marshal(eInfo)
	asynT := asynq.NewTask(f.Name, bs)
	return e.cli.Enqueue(asynT, nil)
}

func (e *Executor) getLeavesResult(sessID string) (map[string]*ExecResult, error) {
	f := e.f
	resMap := map[string]*ExecResult{}
	leaves := f.DAG.GetLeaves()
	for _, leaf := range leaves {
		node := leaf.(*FlowNode) // nolint
		execID := newExecID(f.Name, node.name, sessID)
		eRes, err := e.getExecResult(execID)
		if err != nil {
			return nil, err
		}
		resMap[node.name] = eRes
	}
	return resMap, nil
}

func (e *Executor) setExecResult(eRes *ExecResult) error {
	bs, err := json.Marshal(eRes)
	if err != nil {
		return err
	}
	return e.f.stor.Set(context.TODO(), eRes.ID, bs)
}

func (e *Executor) getExecResult(execID string) (*ExecResult, error) {
	f := e.f
	bs, err := f.stor.Get(context.TODO(), execID)
	if err != nil {
		return nil, err
	}
	if bs == nil {
		return nil, nil
	}
	eRes := &ExecResult{}
	if err := json.Unmarshal(bs, eRes); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal json for ExecResult")
	}
	return eRes, nil
}

func newExecID(flowName, nodeName, randomId string) string {
	return fmt.Sprintf("%s:%s:%s", flowName, nodeName, randomId)
}

func parseExecID(execID string) (flowName string, nodeName string, sessID string, err error) {
	parts := strings.Split(execID, ":")
	if len(parts) != 3 {
		err = errors.Newf("failed to parse execution id")
		return
	}
	flowName, nodeName, sessID = parts[0], parts[1], parts[2]
	return
}
