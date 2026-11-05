package proxy

import (
	"bytes"
	"sync"
	"sync/atomic"
	"testing"
)

func TestCreateAccountDoesNotOverwrite(t *testing.T) {
	Path() // Initialize before substituting an isolated directory.
	old := proxyDir
	proxyDir = t.TempDir()
	t.Cleanup(func() { proxyDir = old })
	sqlite, err := NewAuthMTSQLite3()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlite.Close()
	for name, ab := range map[string]AuthBackend{"files": AuthFiles{}, "sqlite": sqlite} {
		t.Run(name, func(t *testing.T) {
			var nOk atomic.Int32
			var wg sync.WaitGroup
			for i := 0; i < 12; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					if ab.CreateAccount("user", []byte("salt"), []byte("verifier")) == nil {
						nOk.Add(1)
					}
				}()
			}
			wg.Wait()
			if nOk.Load() != 1 {
				t.Fatalf("successful registrations = %d, want 1", nOk.Load())
			}
			// Includes a client that received FirstSRP before panel registration.
			if err := ab.CreateAccount("user", []byte("othersalt"), []byte("otherverifier")); err == nil {
				t.Fatal("duplicate registration accepted")
			}
			salt, verifier, err := ab.Passwd("user")
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(salt, []byte("salt")) || !bytes.Equal(verifier, []byte("verifier")) {
				t.Fatal("credentials overwritten")
			}
		})
	}
}
