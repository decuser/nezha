// Package position provides syntax source positions and line-oriented diagnostic rendering.
package position

// Position represents a source location.
type Position struct {
	File string
	Line int
	Col  int
}
