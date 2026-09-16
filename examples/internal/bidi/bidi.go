package bidi

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/coder/websocket"
)

type Client struct {
	ws      *websocket.Conn
	session string
	nextID  atomic.Int64
	pending map[int64]chan bidiMsg
	mu      sync.Mutex
	done    chan struct{}
	cancel  context.CancelFunc
}

type bidiMsg struct {
	Result json.RawMessage
	Err    *Error
}

type Error struct {
	Code       string `json:"error"`
	Message    string `json:"message"`
	Stacktrace string `json:"stacktrace"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("bidi: %s: %s", e.Code, e.Message)
}

type StatusResponse struct {
	Ready   bool   `json:"ready"`
	Message string `json:"message"`
}

type BrowserContext struct {
	Context  string           `json:"context"`
	URL      string           `json:"url"`
	Parent   *string          `json:"parent"`
	Children []BrowserContext `json:"children"`
}

type RemoteValue struct {
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

func Connect(ctx context.Context, addr string) (*Client, error) {
	body := map[string]any{
		"capabilities": map[string]any{
			"alwaysMatch": map[string]any{
				"webSocketUrl": true,
			},
		},
	}
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	sessionURL := strings.TrimRight(addr, "/") + "/session"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sessionURL, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("session create: %d %s", resp.StatusCode, b)
	}

	var sessionResp struct {
		Value struct {
			SessionID    string `json:"sessionId"`
			Capabilities struct {
				WebSocketURL string `json:"webSocketUrl"`
			} `json:"capabilities"`
		} `json:"value"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&sessionResp); err != nil {
		return nil, err
	}

	wsURL := sessionResp.Value.Capabilities.WebSocketURL
	if wsURL == "" {
		return nil, errors.New("bidi: server did not return webSocketUrl")
	}

	dialCtx, cancel := context.WithCancel(ctx)
	wsConn, _, err := websocket.Dial(dialCtx, wsURL, &websocket.DialOptions{
		Subprotocols: []string{"webdriver.bidi"},
	})
	if err != nil {
		cancel()
		return nil, fmt.Errorf("ws dial: %w", err)
	}

	c := &Client{
		ws:      wsConn,
		session: sessionResp.Value.SessionID,
		pending: make(map[int64]chan bidiMsg),
		done:    make(chan struct{}),
		cancel:  cancel,
	}
	go c.readerLoop()
	return c, nil
}

func (c *Client) readerLoop() {
	defer close(c.done)
	for {
		_, data, err := c.ws.Read(context.Background())
		if err != nil {
			return
		}
		var raw struct {
			ID         int64           `json:"id"`
			Result     json.RawMessage `json:"result"`
			Error      string          `json:"error"`
			Message    string          `json:"message"`
			Stacktrace string          `json:"stacktrace"`
		}
		if err := json.Unmarshal(data, &raw); err != nil || raw.ID == 0 {
			continue
		}
		c.mu.Lock()
		ch := c.pending[raw.ID]
		c.mu.Unlock()
		if ch != nil {
			msg := bidiMsg{Result: raw.Result}
			if raw.Error != "" {
				msg.Err = &Error{Code: raw.Error, Message: raw.Message, Stacktrace: raw.Stacktrace}
			}
			ch <- msg
		}
	}
}

func (c *Client) command(ctx context.Context, method string, params any, out any) error {
	id := c.nextID.Add(1)
	frame := map[string]any{"id": id, "method": method, "params": params}
	data, err := json.Marshal(frame)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	ch := make(chan bidiMsg, 1)
	c.mu.Lock()
	c.pending[id] = ch
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
	}()

	if err := c.ws.Write(ctx, websocket.MessageText, data); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c.done:
		return errors.New("bidi: connection closed")
	case msg := <-ch:
		if msg.Err != nil {
			return msg.Err
		}
		if out != nil && msg.Result != nil {
			if err := json.Unmarshal(msg.Result, out); err != nil {
				return fmt.Errorf("decode: %w", err)
			}
		}
		return nil
	}
}

func (c *Client) Status(ctx context.Context) (*StatusResponse, error) {
	var out StatusResponse
	if err := c.command(ctx, "session.status", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) Contexts(ctx context.Context) ([]BrowserContext, error) {
	var out struct {
		Contexts []BrowserContext `json:"contexts"`
	}
	if err := c.command(ctx, "browsingContext.getTree", nil, &out); err != nil {
		return nil, err
	}
	return out.Contexts, nil
}

func (c *Client) NewPage(ctx context.Context) (string, error) {
	var out struct {
		Context string `json:"context"`
	}
	if err := c.command(ctx, "browsingContext.create", map[string]string{"type": "tab"}, &out); err != nil {
		return "", err
	}
	return out.Context, nil
}

func (c *Client) Navigate(ctx context.Context, contextID, url string) error {
	return c.command(ctx, "browsingContext.navigate", map[string]any{
		"context": contextID,
		"url":     url,
		"wait":    "complete",
	}, nil)
}

func (c *Client) Evaluate(ctx context.Context, contextID, expression string) (*RemoteValue, error) {
	var out struct {
		Result RemoteValue `json:"result"`
	}
	if err := c.command(ctx, "script.evaluate", map[string]any{
		"expression":      expression,
		"target":          map[string]string{"context": contextID},
		"awaitPromise":    true,
		"resultOwnership": "none",
	}, &out); err != nil {
		return nil, err
	}
	return &out.Result, nil
}

func (c *Client) Screenshot(ctx context.Context, contextID string, full bool) ([]byte, error) {
	params := map[string]any{
		"context": contextID,
		"origin":  "viewport",
	}
	if full {
		params["captureBeyondViewport"] = true
	}
	var out struct {
		Data string `json:"data"`
	}
	if err := c.command(ctx, "browsingContext.captureScreenshot", params, &out); err != nil {
		return nil, err
	}
	return base64.StdEncoding.DecodeString(out.Data)
}

func (c *Client) ClosePage(ctx context.Context, contextID string) error {
	return c.command(ctx, "browsingContext.close", map[string]string{"context": contextID}, nil)
}

func (c *Client) Close(ctx context.Context) error {
	err := c.command(ctx, "session.end", nil, nil)
	e2 := c.ws.CloseNow()
	<-c.done
	if err != nil {
		return err
	}
	return e2
}
