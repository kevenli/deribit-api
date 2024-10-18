package deribit

import (
	"os"
	"sync"
	"testing"

	"github.com/frankrap/deribit-api/models"
)

func getConfig() *Configuration {
	cfg := &Configuration{
		Addr:          TestBaseURL,
		ApiKey:        "AsJTU16U",
		SecretKey:     "mM5_K8LVxztN6TjjYpv_cJVGQBvk4jglrEpqkw1b87U",
		AutoReconnect: true,
		DebugMode:     true,
	}

	apiKey := os.Getenv("DERIBIT_API_KEY")
	if apiKey != "" {
		cfg.ApiKey = apiKey
	}

	apiSecret := os.Getenv("DERIBIT_API_SECRET")
	if apiSecret != "" {
		cfg.SecretKey = apiSecret
	}

	return cfg
}

func TestConnectUserWebsocket(t *testing.T) {
	cfg := getConfig()
	websocket, err := NewUserWebsocket(cfg)
	if err != nil {
		t.Fatalf("%s, %v", err, websocket)
	}

	if got, expect := websocket.IsConnected(), true; expect != got {
		t.Errorf("UserWebsocket.IsConnected, got %v != expect %v", got, expect)
	}
}

func TestClientConnectUserWebsocket(t *testing.T) {
	cfg := getConfig()
	client := New(cfg)
	websocket, err := client.ConnectUserWebsocket()
	if err != nil {
		t.Fatalf("%s, %v", err, websocket)
	}

	if got, expect := websocket.IsConnected(), true; expect != got {
		t.Errorf("UserWebsocket.IsConnected, got %v != expect %v", got, expect)
	}
}

func TestConnectUserWebsocketInvalidKey(t *testing.T) {
	cfg := getConfig()
	cfg.SecretKey = "SomeWrongKey"
	client := New(cfg)
	_, err := client.ConnectUserWebsocket()
	if err == nil || err.Error() != "Authenticate Failed" {
		t.Fatalf("Should catch error here.")
	}
}

func TestUserOrderEventSubscription(t *testing.T) {
	cfg := getConfig()
	client := New(cfg)
	websocket, err := client.ConnectUserWebsocket()

	if err != nil {
		t.Fatalf("%s, %v", err, websocket)
	}

	// wait event, notify when any UserOrder event received.
	wg := &sync.WaitGroup{}

	// subscribe on non-existing currency will not get an error
	websocket.SubscribeUserOrders("any", "any", func(o *models.Order) {
		t.Logf("Order event: %v", o)
		wg.Done()
	})

	go websocket.StartConsumeEvents()

	params := &models.BuyParams{
		InstrumentName: "BTC-PERPETUAL",
		Amount:         10,
		Price:          24000,
		Type:           "limit",
	}

	wg.Add(1)
	result, err := client.Buy(params)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("OrderResponse: %v", result)

	// cancel order, ignore code 11020 max_open_orders_per_instrument error
	orderId := result.Order.OrderID
	cancelResult, err := client.Cancel(&models.CancelParams{OrderID: orderId})
	if err != nil {
		t.Fatal(err, cancelResult)
	}

	wg.Wait()
}

// func TestWrongUserOrdersSubscription(t *testing.T) {
// 	cfg := getConfig()
// 	client := New(cfg)
// 	websocket, err := client.ConnectUserWebsocket()

// 	if err != nil {
// 		t.Fatalf("%s, %v", err, websocket)
// 	}

// 	// subscribe on non-existing currency will not get an error
// 	ch := make(chan *models.Order)
// 	websocket.SubscribeUserOrders("any", "wrong_currency", func(o *models.Order) {
// 		t.Logf("Order event: %v", o)
// 		ch <- o
// 	})

// 	err = websocket.StartConsumeEvents()
// 	if err != nil {
// 		t.Fatalf("Subscription error not caught")
// 	}

// 	<-ch
// }
