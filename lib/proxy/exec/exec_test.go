package gexec

import (
	"context"
	"slices"
	"testing"

	"github.com/rumpelsepp/gcat/lib/proxy"
)

func TestSpawn(t *testing.T) {
	addr, err := proxy.ParseAddr("exec:?cmd=cat /dev/zero")
	if err != nil {
		t.Fatal(err)
	}

	p, err := proxy.Registry.FindAndCreateProxy(addr)
	if err != nil {
		t.Fatal(err)
	}

	conn, err := p.Connect(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	buf := make([]byte, 64)

	if _, err := conn.Read(buf); err != nil {
		t.Fatal(err)
	}
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}

	expected := make([]byte, 64)

	if !slices.Equal(buf, expected) {
		t.Fatalf("got unexpected data: len()=%d repr()=%+v", len(expected), expected)
	}
}
