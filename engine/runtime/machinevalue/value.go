// Package machinevalue owns the interned fixed-width scalar values used at
// machine/realization boundaries. The finite domains are allocated once so hot
// register/address traffic does not allocate ordinary Nezha values.
package machinevalue

import "nezha/engine/semantics/value"

const Provider = "machine/ffi"

var bytes [1 << 8]value.Opaque
var words [1 << 16]value.Opaque
var addrs18 [1 << 18]value.Opaque

func init() {
	for i := range bytes {
		bytes[i] = value.Opaque{Provider: Provider, Kind: "byte", Bits: uint32(i)}
	}
	for i := range words {
		words[i] = value.Opaque{Provider: Provider, Kind: "word", Bits: uint32(i)}
	}
	for i := range addrs18 {
		addrs18[i] = value.Opaque{Provider: Provider, Kind: "addr18", Bits: uint32(i)}
	}
}

func Word(v uint32) value.Value   { return &words[v&0xffff] }
func Byte(v uint32) value.Value   { return &bytes[v&0xff] }
func Addr18(v uint32) value.Value { return &addrs18[v&0x3ffff] }

func Bits(v value.Value, kind string) (uint32, bool) {
	o, ok := v.(*value.Opaque)
	if !ok || o.Provider != Provider || o.Kind != kind {
		return 0, false
	}
	return o.Bits, true
}
