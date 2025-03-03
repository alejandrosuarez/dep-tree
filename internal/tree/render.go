package tree

import (
	"strconv"

	"github.com/gabotechs/dep-tree/internal/board"
	"github.com/gabotechs/dep-tree/internal/utils"
)

const indent = 2
const NodeIdTag = "nodeId"
const NodeIndexTag = "nodeIndex"
const ConnectorOriginNodeIdTag = "connectorOrigin"
const ConnectorDestinationNodeIdTag = "connectorDestination"
const NodeFromTag = "nodeFrom"

func (t *Tree[T]) Render() (*board.Board, error) {
	b := board.MakeBoard()

	lastLevel := -1
	prefix := ""
	xOffsetCount := 0
	xOffset := 0
	yOffset := 0
	for i, n := range t.Nodes {
		if n.Lvl == lastLevel {
			if len(t.Graph.FromId(t.Nodes[i-1].Node.Id)) > 0 {
				xOffsetCount++
				prefix += " "
			}
		} else {
			lastLevel = n.Lvl
			xOffset += xOffsetCount
			xOffsetCount = 0
			prefix = ""
			if i != 0 {
				yOffset++
			}
		}

		fromNodes := t.Graph.ToId(n.Node.Id)

		tags := map[string]string{
			NodeIdTag:    n.Node.Id,
			NodeIndexTag: strconv.Itoa(i),
			NodeFromTag:  "",
		}

		for _, from := range fromNodes {
			tags[NodeFromTag] += from.Id + ";"
		}

		err := b.AddBlock(
			&board.Block{
				Id:       n.Node.Id,
				Label:    prefix + t.NodeParser.Display(n.Node).Name,
				Position: utils.Vec(indent*n.Lvl+xOffset, i+yOffset),
				Tags:     tags,
			},
		)
		if err != nil {
			return nil, err
		}
	}

	for _, n := range t.Nodes {
		for _, to := range t.Graph.FromId(n.Node.Id) {
			tags := map[string]string{
				ConnectorOriginNodeIdTag:      n.Node.Id,
				ConnectorDestinationNodeIdTag: to.Id,
			}

			err := b.AddConnector(n.Node.Id, to.Id, tags)
			if err != nil {
				return nil, err
			}
		}
	}
	for _, cycle := range t.Cycles.Keys() {
		tags := map[string]string{
			ConnectorOriginNodeIdTag:      cycle[0],
			ConnectorDestinationNodeIdTag: cycle[1],
		}

		err := b.AddConnector(cycle[0], cycle[1], tags)
		if err != nil {
			return nil, err
		}
	}
	return b, nil
}
