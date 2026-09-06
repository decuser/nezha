package hostnative

import (
	"testing"
)

func TestTransportDirectBindingUsesPackedWordAndLocalOffset(t *testing.T) {
	misses := 0
	transport, err := NewTransport(18, func(offset uint32, packet Word) Word {
		misses++
		return packet ^ Word(offset)
	})
	if err != nil {
		t.Fatal(err)
	}
	id, err := transport.RegisterDirect(01000, 01007, func(offset uint32, packet Word) Word {
		return packet ^ Word(offset+1)
	})
	if err != nil {
		t.Fatal(err)
	}

	const packet Word = 0xfedcba9876543210
	out := transport.Transfer(01003, packet)
	if out.Yield {
		t.Fatal("direct endpoint yielded")
	}
	if out.Endpoint != id || out.Offset != 3 {
		t.Fatalf("route = endpoint %d offset %d, want endpoint %d offset 3", out.Endpoint, out.Offset, id)
	}
	if out.Packet != packet^4 {
		t.Fatalf("response = %#x, want %#x", out.Packet, packet^4)
	}
	if misses != 0 {
		t.Fatalf("miss endpoint called %d times", misses)
	}

	miss := transport.Transfer(0777, packet)
	if miss.Endpoint != 0 || miss.Offset != 0777 || miss.Yield {
		t.Fatalf("miss route = %+v", miss)
	}
	if misses != 1 {
		t.Fatalf("miss endpoint called %d times, want 1", misses)
	}
}

func TestTransportInterpretedBindingYieldsResolvedPacket(t *testing.T) {
	transport, err := NewTransport(18, func(offset uint32, packet Word) Word { return packet })
	if err != nil {
		t.Fatal(err)
	}
	id, err := transport.RegisterInterpreted(0172520, 0172532)
	if err != nil {
		t.Fatal(err)
	}

	const packet Word = 0x00ff112233445566
	out := transport.Transfer(0172526, packet)
	if !out.Yield {
		t.Fatal("interpreted endpoint did not yield")
	}
	if out.Endpoint != id || out.Offset != 6 || out.Packet != packet {
		t.Fatalf("yield = %+v, want endpoint %d offset 6 packet %#x", out, id, packet)
	}
}

func TestTransportRejectsOverlappingRegistration(t *testing.T) {
	transport, err := NewTransport(18, func(offset uint32, packet Word) Word { return packet })
	if err != nil {
		t.Fatal(err)
	}
	if _, err := transport.RegisterDirect(01000, 01007, func(offset uint32, packet Word) Word { return packet }); err != nil {
		t.Fatal(err)
	}
	if _, err := transport.RegisterInterpreted(01007, 01010); err == nil {
		t.Fatal("overlapping endpoint registration accepted")
	}
}

func TestTransportDirectPathAllocatesNothing(t *testing.T) {
	transport, err := NewTransport(18, func(offset uint32, packet Word) Word { return packet })
	if err != nil {
		t.Fatal(err)
	}
	_, err = transport.RegisterDirect(01000, 01007, func(offset uint32, packet Word) Word {
		return packet + Word(offset)
	})
	if err != nil {
		t.Fatal(err)
	}

	var sink Word
	allocs := testing.AllocsPerRun(1000, func() {
		out := transport.Transfer(01003, 0x8000000000000001)
		sink ^= out.Packet
	})
	if allocs != 0 {
		t.Fatalf("direct transfer allocations = %v, want 0", allocs)
	}
	_ = sink
}

func BenchmarkTransportDirect(b *testing.B) {
	transport, err := NewTransport(18, func(offset uint32, packet Word) Word { return packet })
	if err != nil {
		b.Fatal(err)
	}
	_, err = transport.RegisterDirect(01000, 01007, func(offset uint32, packet Word) Word {
		return packet + Word(offset)
	})
	if err != nil {
		b.Fatal(err)
	}
	var sink Word
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out := transport.Transfer(01003, Word(i))
		sink ^= out.Packet
	}
	_ = sink
}

func TestTransportUnregisterRestoresMiss(t *testing.T) {
	misses := 0
	transport, err := NewTransport(8, func(offset uint32, packet Word) Word {
		misses++
		return packet + Word(offset)
	})
	if err != nil {
		t.Fatal(err)
	}
	id, err := transport.RegisterDirect(10, 12, func(offset uint32, packet Word) Word { return 0xabc + Word(offset) })
	if err != nil {
		t.Fatal(err)
	}
	if got := transport.Transfer(11, 0); got.Packet != 0xabd {
		t.Fatalf("direct = %+v", got)
	}
	if err := transport.Unregister(id); err != nil {
		t.Fatal(err)
	}
	got := transport.Transfer(11, 7)
	if got.Endpoint != 0 || got.Packet != 18 || misses != 1 {
		t.Fatalf("miss = %+v misses=%d", got, misses)
	}
}

func TestTransportReplaceDirectRefinesInterpretedRange(t *testing.T) {
	transport, err := NewTransport(8, func(offset uint32, packet Word) Word { return 0 })
	if err != nil {
		t.Fatal(err)
	}
	if _, err := transport.RegisterInterpreted(100, 119); err != nil {
		t.Fatal(err)
	}
	id, err := transport.ReplaceDirect(104, 107, func(offset uint32, packet Word) Word {
		return Word(0x1000 + offset)
	})
	if err != nil {
		t.Fatal(err)
	}
	if out := transport.Transfer(103, 7); !out.Yield {
		t.Fatalf("selector 103 no longer interpreted: %+v", out)
	}
	if out := transport.Transfer(105, 7); out.Yield || out.Endpoint != id || out.Offset != 1 || out.Packet != 0x1001 {
		t.Fatalf("replacement route = %+v", out)
	}
	if out := transport.Transfer(108, 7); !out.Yield {
		t.Fatalf("selector 108 no longer interpreted: %+v", out)
	}
}

func TestTransportReplaceDirectRejectsMissAndDirectRoutes(t *testing.T) {
	transport, err := NewTransport(8, func(offset uint32, packet Word) Word { return 0 })
	if err != nil {
		t.Fatal(err)
	}
	if _, err := transport.ReplaceDirect(10, 11, func(uint32, Word) Word { return 0 }); err == nil {
		t.Fatal("replacement of miss routes accepted")
	}
	if _, err := transport.RegisterDirect(20, 21, func(uint32, Word) Word { return 0 }); err != nil {
		t.Fatal(err)
	}
	if _, err := transport.ReplaceDirect(20, 21, func(uint32, Word) Word { return 0 }); err == nil {
		t.Fatal("replacement of direct routes accepted")
	}
}
