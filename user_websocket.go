package deribit

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/frankrap/deribit-api/models"
	"github.com/sourcegraph/jsonrpc2"
	"nhooyr.io/websocket"
)

type UserWebsocket struct {
	ctx       context.Context
	addr      string
	apiKey    string
	secretKey string
	conn      *websocket.Conn
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

func (c *Client) ConnectUserWebsocket() (*UserWebsocket, error) {
	websocket := UserWebsocket{
		ctx:       c.ctx,
		addr:      c.addr,
		apiKey:    c.apiKey,
		secretKey: c.secretKey,
	}
	err := websocket.connect()
	if err != nil {
		return nil, err
	}

	return &websocket, nil
}
