package substrate

import (
	"bufio"
	"fmt"
	"io"

	"nezha/engine/runtime/hal"
	"nezha/engine/semantics/value"
)

func (g *GoRuntime) halPrint(args []value.Value, ctx *hal.EvalContext) value.Value {
	for _, arg := range args {
		if s, ok := arg.(*value.String); ok {
			fmt.Fprint(g.stdout, s.Val)
		} else {
			fmt.Fprint(g.stdout, arg.Inspect())
		}
	}
	return value.EMPTY
}

func (g *GoRuntime) halRead(args []value.Value, ctx *hal.EvalContext) value.Value {
	g.mu.RLock()
	reader := g.stdinReader
	g.mu.RUnlock()
	if reader == nil {
		reader = bufio.NewReader(g.stdin)
	}
	line, err := reader.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			if line == "" {
				return &value.List{Shape: "end", Elements: []value.Value{}}
			}
			return &value.String{Val: line}
		}
		return value.NewShapedError("io", "read: %v", err)
	}
	if len(line) > 0 && line[len(line)-1] == '\n' {
		line = line[:len(line)-1]
	}
	if len(line) > 0 && line[len(line)-1] == '\r' {
		line = line[:len(line)-1]
	}
	return &value.String{Val: line}
}
