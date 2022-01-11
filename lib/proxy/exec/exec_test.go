package exec

import (
	"os/exec"
	"testing"
)

func TestSpawn(t *testing.T) {
	proxy := ProxyExec{
		Command: exec.Command("cat", "/dev/urandom"),
	}

	p, err := proxy.Dial()
	if err != nil {
		t.Fatal(err)
	}

	buf := make([]byte, 64)

	if _, err := p.Read(buf); err != nil {
		t.Fatal(err)
	}
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}
}
