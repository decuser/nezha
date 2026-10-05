package substrate

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"nezha/engine/semantics/value"
)

func (g *GoRuntime) halPrint(args []value.Value) value.Value {
	for _, arg := range args {
		if s, ok := arg.(*value.String); ok {
			fmt.Fprint(g.stdout, s.Val)
		} else {
			fmt.Fprint(g.stdout, arg.Inspect())
		}
	}
	return value.EMPTY
}

func (g *GoRuntime) halRead(args []value.Value) value.Value {
	line, err := g.stdin.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return value.NewFault("read: %v", err)
	}
	if errors.Is(err, io.EOF) && len(line) == 0 {
		return &value.String{Val: ""}
	}
	line = strings.TrimSuffix(line, "\n")
	line = strings.TrimSuffix(line, "\r")
	return &value.String{Val: line}
}
