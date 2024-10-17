package deribit

import (
	"testing"
)

func TestConnectUserWebsocket(t *testing.T) {
	cfg := &Configuration{
		Addr:          TestBaseURL,
		ApiKey:        "AsJTU16U",
		SecretKey:     "mM5_K8LVxztN6TjjYpv_cJVGQBvk4jglrEpqkw1b87U",
		AutoReconnect: true,
		DebugMode:     true,
	}
	client := New(cfg)
	websocket, err := client.ConnectUserWebsocket()
	if err != nil {
		t.Fatalf("%s, %v", err, websocket)
	}
}

func TestConnectUserWebsocketInvalidKey(t *testing.T) {
	cfg := &Configuration{
		Addr:          TestBaseURL,
		ApiKey:        "AsJTU16U",
		SecretKey:     "xxx",
		AutoReconnect: true,
		DebugMode:     true,
	}
	client := New(cfg)
	_, err := client.ConnectUserWebsocket()
	if err == nil || err.Error() != "Authenticate Failed" {
		t.Fatalf("Should catch error here.")
	}
}
