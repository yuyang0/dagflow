package flow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDAG(t *testing.T) {
	// TODO
	f := New("flow1", nil, nil, nil, nil, nil)
	f.Node("l1n1", nil)
	f.Node("l2n1", nil)
	f.Node("l2n2", nil)
	f.Node("l3n1", nil)
	f.Edge("l1n1", "l2n1")
	f.Edge("l1n1", "l2n2")
	f.Edge("l2n1", "l3n1")
	f.Edge("l2n2", "l3n1")
	roots := f.DAG.GetRoots()
	assert.Len(t, roots, 1)
	for k := range roots {
		assert.Equal(t, k, "l1n1")
	}
	children, err := f.DAG.GetChildren("l1n1")
	assert.Nil(t, err)
	assert.Len(t, children, 2)
	for k, v := range children {
		node, ok := v.(*FlowNode)
		assert.True(t, ok)
		assert.Equal(t, k, node.name)
		assert.True(t, k == "l2n1" || k == "l2n2")
	}
	parents, err := f.DAG.GetParents("l3n1")
	assert.Nil(t, err)
	assert.Len(t, parents, 2)

	for k, v := range parents {
		node, ok := v.(*FlowNode)
		assert.True(t, ok)
		assert.Equal(t, k, node.name)
		assert.True(t, k == "l2n1" || k == "l2n2")
	}
	children, err = f.DAG.GetChildren("l3n1")
	assert.Nil(t, err)
	assert.Len(t, children, 0)
}
