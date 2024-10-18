package deribit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/chuckpreslar/emission"
	"github.com/frankrap/deribit-api/models"
	"github.com/sourcegraph/jsonrpc2"
	"nhooyr.io/websocket"
)

type UserOrdersEventCallback func(*models.Order)

// type UserWebsocketCallbacks struct {
// 	userOrdersCallback UserOrdersEventCallback
// }

type UserWebsocket struct {
	ctx         context.Context
	addr        string
	apiKey      string
	secretKey   string
	conn        *websocket.Conn
	isConnected bool
	mu          sync.RWMutex
	// callbacks   UserWebsocketCallbacks
	emitter *emission.Emitter
}

func NewUserWebsocket(cfg *Configuration) (*UserWebsocket, error) {
	ctx := cfg.Ctx
	if ctx == nil {
		ctx = context.Background()
	}
	websocket := &UserWebsocket{
		ctx:       ctx,
		addr:      cfg.Addr,
		apiKey:    cfg.ApiKey,
		secretKey: cfg.SecretKey,
		emitter:   emission.NewEmitter(),
	}
	err := websocket.connect()
	if err != nil {
		return nil, err
	}

	err = websocket.authenticate()
	if err != nil {
		return nil, err
	}

	return websocket, nil
}

func (u *UserWebsocket) connect() error {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	conn, _, err := websocket.Dial(ctx, u.addr, &websocket.DialOptions{})
	if err != nil {
		return err
	}
	conn.SetReadLimit(32768 * 64)

	u.conn = conn
	err = u.authenticate()
	if err != nil {
		return err
	}
	u.setIsConnected(true)
	return nil
}

func (u *UserWebsocket) authenticate() error {
	params := models.ClientCredentialsParams{
		GrantType:    "client_credentials",
		ClientID:     u.apiKey,
		ClientSecret: u.secretKey,
	}

	request := jsonrpc2.Request{Method: "/public/auth"}
	request.SetParams(params)
	data, _ := json.Marshal(request)
	u.conn.Write(u.ctx, websocket.MessageText, data)

	response := jsonrpc2.Response{}
	_, recvData, err := u.conn.Read(u.ctx)
	if err != nil {
		return err
	}

	err = json.Unmarshal(recvData, &response)

	if err != nil {
		return err
	}

	// auth error: jsonrpc2: code 13004 message: invalid_credentials
	if response.Error != nil {
		return response.Error
	}

	res := models.AuthResponse{}
	err = json.Unmarshal(*response.Result, &res)
	if err != nil {
		return err
	}

	if res.AccessToken == "" {
		return errors.New("Authenticate Failed")
	}

	return nil
}

// Get UserWebsocket object from an existing client, which is also assured connected.
// This method is not recommanded now, because Client has already established
// a websocket connection.
// This is useful when Client only provides restful apis, and user want
// more features about streaming.
func (c *Client) ConnectUserWebsocket() (*UserWebsocket, error) {
	websocket := UserWebsocket{
		ctx:       c.ctx,
		addr:      c.addr,
		apiKey:    c.apiKey,
		secretKey: c.secretKey,
		emitter:   emission.NewEmitter(),
	}
	err := websocket.connect()
	if err != nil {
		return nil, err
	}

	return &websocket, nil
}

// setIsConnected sets state for isConnected
func (u *UserWebsocket) setIsConnected(state bool) {
	u.mu.Lock()
	defer u.mu.Unlock()

	u.isConnected = state
}

// IsConnected returns the WebSocket connection state
func (u *UserWebsocket) IsConnected() bool {
	u.mu.RLock()
	defer u.mu.RUnlock()

	return u.isConnected
}

func (u *UserWebsocket) SubscribeUserOrders(kind string, currency string, callback UserOrdersEventCallback) error {
	request := jsonrpc2.Request{
		ID:     jsonrpc2.ID{Num: 1},
		Method: "/private/subscribe",
	}

	channel := fmt.Sprintf("user.orders.%s.%s.raw", kind, currency)
	params := models.SubscribeParams{}
	params.Channels = []string{channel}

	request.SetParams(params)
	sendData, err := json.Marshal(request)
	if err != nil {
		return err
	}

	err = u.conn.Write(u.ctx, websocket.MessageText, sendData)
	if err != nil {
		return err
	}

	// u.callbacks.userOrdersCallback = callback
	u.emitter.On(channel, callback)
	return nil
}

// This method will loop receiving events until an error occured.
func (u *UserWebsocket) StartConsumeEvents() error {
	for {
		response := jsonrpc2.Response{}
		event := Event{}

		_, recvData, err := u.conn.Read(u.ctx)
		if err != nil {
			return err
		}

		// jsonrpc downstream message will be either a Response or
		// a Notification(as structure of Ruquest)
		// see: https://www.jsonrpc.org/specification
		// detect if it is a Response
		if err := json.Unmarshal(recvData, &response); err != nil {
			return err
		}

		if response.ID.Num != 0 || response.ID.Str != "" {
			// No id value, data is not a response.
			log.Printf("Response received, %v", response)

			if response.Error != nil {
				return response.Error
			}
			continue
		}

		// notification data should be a Request.
		request := jsonrpc2.Request{}
		json.Unmarshal(recvData, &request)
		if request.Method != "subscription" {
			log.Printf("non-subscription event received, %v", request)
			continue
		}

		if err := json.Unmarshal(*request.Params, &event); err != nil {
			return err
		}

		if strings.HasPrefix(event.Channel, "user.orders") {
			order := models.Order{}
			if err := json.Unmarshal(event.Data, &order); err != nil {
				return err
			}

			// if u.callbacks.userOrdersCallback != nil {
			// 	u.callbacks.userOrdersCallback(&order)
			// }
			u.emitter.Emit(event.Channel, &order)
		} else {
			log.Printf("Unknown event %s", event.Channel)
		}
	}
}
