package gowebsocket

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	logging "github.com/sacOO7/go-logger"
)

type empty struct {
}

var logger = logging.GetLogger(reflect.TypeOf(empty{}).PkgPath()).SetLevel(logging.OFF)

// EnableLogging enables the logger
func (socket Socket) EnableLogging() {
	logger.SetLevel(logging.TRACE)
}

// GetLogger gets the logger object
func (socket Socket) GetLogger() logging.Logger {
	return logger
}

// Socket provides a websocket request
type Socket struct {
	Conn              *websocket.Conn
	WebsocketDialer   *websocket.Dialer
	URL               string
	ConnectionOptions ConnectionOptions
	RequestHeader     http.Header
	OnConnected       func(socket Socket)
	OnTextMessage     func(message string, socket Socket)
	OnBinaryMessage   func(data []byte, socket Socket)
	OnConnectError    func(err error, socket Socket)
	OnDisconnected    func(err error, socket Socket)
	OnPingReceived    func(data string, socket Socket)
	OnPongReceived    func(data string, socket Socket)
	IsConnected       bool
	Timeout           time.Duration
	sendMu            *sync.Mutex // Prevent "concurrent write to websocket connection"
	receiveMu         *sync.Mutex
}

// ConnectionOptions contains connection options
type ConnectionOptions struct {
	UseCompression bool
	UseSSL         bool
	Proxy          func(*http.Request) (*url.URL, error)
	Subprotocols   []string
}

// ReconnectionOptions provides options for reconnecting to the websocket
// TODO Yet to be done
type ReconnectionOptions struct {
}

// New creates a new websocket for the given url
func New(url string) Socket {
	return Socket{
		URL:           url,
		RequestHeader: http.Header{},
		ConnectionOptions: ConnectionOptions{
			UseCompression: false,
			UseSSL:         true,
		},
		WebsocketDialer: &websocket.Dialer{},
		Timeout:         0,
		sendMu:          &sync.Mutex{},
		receiveMu:       &sync.Mutex{},
	}
}

func (socket *Socket) setConnectionOptions() {
	socket.WebsocketDialer.EnableCompression = socket.ConnectionOptions.UseCompression
	socket.WebsocketDialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: socket.ConnectionOptions.UseSSL}
	socket.WebsocketDialer.Proxy = socket.ConnectionOptions.Proxy
	socket.WebsocketDialer.Subprotocols = socket.ConnectionOptions.Subprotocols
}

// Connect connects to the websocket server
func (socket *Socket) Connect() {
	var err error
	var resp *http.Response
	socket.setConnectionOptions()

	socket.Conn, resp, err = socket.WebsocketDialer.Dial(socket.URL, socket.RequestHeader)

	if err != nil {
		logger.Error.Println("Error while connecting to server ", err)
		logger.Error.Println(fmt.Sprintf("HTTP Response %d status: %s", resp.StatusCode, resp.Status))
		socket.IsConnected = false
		if socket.OnConnectError != nil {
			socket.OnConnectError(err, *socket)
		}
		return
	}

	logger.Info.Println("Connected to server")

	if socket.OnConnected != nil {
		socket.IsConnected = true
		socket.OnConnected(*socket)
	}

	defaultPingHandler := socket.Conn.PingHandler()
	socket.Conn.SetPingHandler(func(appData string) error {
		logger.Trace.Println("Received PING from server")
		if socket.OnPingReceived != nil {
			socket.OnPingReceived(appData, *socket)
		}
		return defaultPingHandler(appData)
	})

	defaultPongHandler := socket.Conn.PongHandler()
	socket.Conn.SetPongHandler(func(appData string) error {
		logger.Trace.Println("Received PONG from server")
		if socket.OnPongReceived != nil {
			socket.OnPongReceived(appData, *socket)
		}
		return defaultPongHandler(appData)
	})

	defaultCloseHandler := socket.Conn.CloseHandler()
	socket.Conn.SetCloseHandler(func(code int, text string) error {
		result := defaultCloseHandler(code, text)
		logger.Warning.Println("Disconnected from server ", result)
		if socket.OnDisconnected != nil {
			socket.IsConnected = false
			socket.OnDisconnected(errors.New(text), *socket)
		}
		return result
	})

	go func() {
		for {
			socket.receiveMu.Lock()
			if socket.Timeout != 0 {
				err := socket.Conn.SetReadDeadline(time.Now().Add(socket.Timeout))
				if err != nil {
					logger.Error.Println(err)
				}
			}
			messageType, message, err := socket.Conn.ReadMessage()
			socket.receiveMu.Unlock()
			if err != nil {
				logger.Error.Println(fmt.Sprintf("read: %s", err))
				if socket.OnDisconnected != nil {
					socket.IsConnected = false
					socket.OnDisconnected(err, *socket)
				}
				return
			}
			logger.Info.Println(fmt.Sprintf("recv: %s", message))

			switch messageType {
			case websocket.TextMessage:
				if socket.OnTextMessage != nil {
					socket.OnTextMessage(string(message), *socket)
				}
			case websocket.BinaryMessage:
				if socket.OnBinaryMessage != nil {
					socket.OnBinaryMessage(message, *socket)
				}
			}
		}
	}()
}

// SendText sends a test message to the server
func (socket *Socket) SendText(message string) {
	err := socket.send(websocket.TextMessage, []byte(message))
	if err != nil {
		logger.Error.Println("write:", err)
		return
	}
}

// SendBinary sends a binary message to the websocket
func (socket *Socket) SendBinary(data []byte) {
	err := socket.send(websocket.BinaryMessage, data)
	if err != nil {
		logger.Error.Println("write:", err)
		return
	}
}

func (socket *Socket) send(messageType int, data []byte) error {
	socket.sendMu.Lock()
	err := socket.Conn.WriteMessage(messageType, data)
	socket.sendMu.Unlock()
	return err
}

// Close closese the websocket
func (socket *Socket) Close() {
	err := socket.send(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		logger.Error.Println("write close:", err)
	}
	_ = socket.Conn.Close()
	if socket.OnDisconnected != nil {
		socket.IsConnected = false
		socket.OnDisconnected(err, *socket)
	}
}
