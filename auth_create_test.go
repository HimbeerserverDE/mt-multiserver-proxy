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
	for name, backend := range map[string]AuthBackend{"files": AuthFiles{}, "sqlite": sqlite} {
		t.Run(name, func(t *testing.T) {
			var successes atomic.Int32
			var wg sync.WaitGroup
			for i := 0; i < 12; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					if createAccount(backend, "Student", []byte("salt"), []byte("verifier")) == nil {
						successes.Add(1)
					}
				}()
			}
			wg.Wait()
			if successes.Load() != 1 {
				t.Fatalf("successful registrations = %d, want 1", successes.Load())
			}
			// Includes a client that received FirstSRP before panel registration.
			if err := createAccount(backend, "Student", []byte("other"), []byte("other")); err == nil {
				t.Fatal("duplicate registration accepted")
			}
			salt, verifier, err := backend.Passwd("Student")
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(salt, []byte("salt")) || !bytes.Equal(verifier, []byte("verifier")) {
				t.Fatal("credentials overwritten")
			}
		})
	}
}
