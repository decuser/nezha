package syntax

import (
	"aiki/engine"
	"aiki/engine/observe"
)

// Node represents an AST node.
type Node struct {
	Type     string  // production name or token type
	Value    string  // lexeme for terminals
	Children []*Node // for non-terminals
	Pos      engine.Position

	// ObservationBinding caches a profiling-network node identity for this AST
	// object. It is runtime metadata only and does not participate in syntax or
	// static coverage identity.
	ObservationBinding observe.NodeBinding
}

// Child returns the nth child or nil.
func (n *Node) Child(i int) *Node {
	if i < 0 || i >= len(n.Children) {
		return nil
	}
	return n.Children[i]
}

// ChildByType returns the first child with matching type.
func (n *Node) ChildByType(typ string) *Node {
	for _, c := range n.Children {
		if c.Type == typ {
			return c
		}
	}
	return nil
}

// ChildrenByType returns all children with matching type.
func (n *Node) ChildrenByType(typ string) []*Node {
	var result []*Node
	for _, c := range n.Children {
		if c.Type == typ {
			result = append(result, c)
		}
	}
	return result
}

// HasChild returns true if node has a child of the given type.
func (n *Node) HasChild(typ string) bool {
	return n.ChildByType(typ) != nil
}

// IsTerminal returns true if this node is a terminal (has no children).
func (n *Node) IsTerminal() bool {
	return len(n.Children) == 0
}
