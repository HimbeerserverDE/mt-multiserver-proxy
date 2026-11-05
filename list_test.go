package proxy

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAnnounceAddress(t *testing.T) {
	Conf() // load the default config first so it doesn't overwrite ours

	var got map[string]interface{}
	status := http.StatusOK
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = nil
		json.Unmarshal([]byte(r.FormValue("json")), &got)
		w.WriteHeader(status)
		w.Write([]byte("Requester IP does not match host"))
	}))
	defer srv.Close()

	setConf := func(bindAddr, address string, port int) {
		configMu.Lock()
		defer configMu.Unlock()

		config.BindAddr = bindAddr
		config.List.Addr = srv.URL
		config.List.Address = address
		config.List.Port = port
	}

	tests := []struct {
		bindAddr, address string
		port              int
		wantAddr          interface{}
		wantPort          float64
	}{
		{":30000", "", 0, nil, 30000},
		{"0.0.0.0:30000", "", 0, nil, 30000},
		{"192.0.2.1:30000", "", 0, "192.0.2.1", 30000},
		{":30000", "play.example.org", 0, "play.example.org", 30000},
		{":30000", "play.example.org", 40000, "play.example.org", 40000},
	}

	for _, tt := range tests {
		setConf(tt.bindAddr, tt.address, tt.port)
		if err := announce(listUpdate); err != nil {
			t.Fatalf("%+v: %v", tt, err)
		}

		if got["address"] != tt.wantAddr {
			t.Errorf("%+v: address = %v, want %v", tt, got["address"], tt.wantAddr)
		}
		if got["port"] != tt.wantPort {
			t.Errorf("%+v: port = %v, want %v", tt, got["port"], tt.wantPort)
		}
	}

	status = http.StatusAccepted
	if err := announce(listAdd); err != nil {
		t.Errorf("announce failed on %d: %v", status, err)
	}

	status = http.StatusConflict
	if err := announce(listUpdate); err == nil {
		t.Error("announce succeeded on a list server error")
	}
}
