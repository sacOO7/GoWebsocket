package main

import (
	"github.com/gorilla/websocket"
	"net/http"
	"log"
	"os"
	"os/signal"
	"errors"
	"crypto/tls"
	"net/url"
	"sync"
)

type Socket struct {
	conn              *websocket.Conn
	websocketDialer   *websocket.Dialer
	url               string
	connectionOptions ConnectionOptions
	requestHeader     http.Header
	OnConnected       func(socket Socket)
	OnTextMessage     func(message string, socket Socket)
	OnBinaryMessage   func(data [] byte, socket Socket)
	OnConnectError    func(err error, socket Socket)
	OnDisconnected    func(err error, socket Socket)
	OnPingReceived    func(data string, socket Socket)
	OnPongReceived    func(data string, socket Socket)
	isConnected       bool
	sendMu            *sync.Mutex // Prevent "concurrent write to websocket connection"
	receiveMu         *sync.Mutex
}

type ConnectionOptions struct {
	useCompression bool
	reconnect      bool
	useSSL         bool
	proxy          func(*http.Request) (*url.URL, error)
	subprotocols   [] string
}

func New(url string, requestHeader http.Header) Socket {
	return Socket{url: url, requestHeader: requestHeader,
		connectionOptions: ConnectionOptions{
			reconnect:      false,
			useCompression: false,
			useSSL:         true,
		},
		websocketDialer: &websocket.Dialer{},
		sendMu: &sync.Mutex{},
		receiveMu: &sync.Mutex{},
	}
}

func (socket *Socket) setConnectionOptions(options ConnectionOptions) {
	socket.connectionOptions = options
}

func (socket *Socket) IsConnected() bool {
	return socket.isConnected
}

func (socket *Socket) registerConnectionOptions() {
	socket.websocketDialer.EnableCompression = socket.connectionOptions.useCompression
	socket.websocketDialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: socket.connectionOptions.useSSL}
	socket.websocketDialer.Proxy = socket.connectionOptions.proxy
	socket.websocketDialer.Subprotocols = socket.connectionOptions.subprotocols
}

func (socket *Socket) Connect() {
	var err error;
	socket.registerConnectionOptions()

	socket.conn, _, err = socket.websocketDialer.Dial(socket.url, socket.requestHeader)

	if err != nil {
		socket.isConnected = false
		if socket.OnConnectError != nil {
			socket.OnConnectError(err, *socket)
		}
		return
	}

	if socket.OnConnected != nil {
		socket.isConnected = true
		socket.OnConnected(*socket)
	}

	defaultPingHandler := socket.conn.PingHandler()
	socket.conn.SetPingHandler(func(appData string) error {
		if socket.OnPingReceived != nil {
			socket.OnPingReceived(appData, *socket)
		}
		return defaultPingHandler(appData)
	})

	defaultPongHandler := socket.conn.PongHandler()
	socket.conn.SetPongHandler(func(appData string) error {
		if socket.OnPongReceived != nil {
			socket.OnPongReceived(appData, *socket)
		}
		return defaultPongHandler(appData)
	})

	defaultCloseHandler := socket.conn.CloseHandler()
	socket.conn.SetCloseHandler(func(code int, text string) error {
		result := defaultCloseHandler(code, text)
		if socket.OnDisconnected != nil {
			socket.isConnected = false
			socket.OnDisconnected(errors.New(text), *socket)
		}
		return result
	})

	go func() {
		for {
			socket.receiveMu.Lock()
			messageType, message, err := socket.conn.ReadMessage()
			socket.receiveMu.Unlock()
			if err != nil {
				log.Println("read:", err)
				return
			}
			log.Printf("recv: %s", message)

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

func (socket *Socket) SendText(message string) {
	socket.sendMu.Lock()
	err := socket.conn.WriteMessage(websocket.TextMessage, [] byte (message))
	socket.sendMu.Unlock()
	if err != nil {
		log.Println("write:", err)
		return
	}
}

func (socket *Socket) SendBinary(data [] byte) {
	socket.sendMu.Lock()
	err := socket.conn.WriteMessage(websocket.BinaryMessage, data)
	socket.sendMu.Unlock()
	if err != nil {
		log.Println("write:", err)
		return
	}
}

func (socket *Socket) Close() {
	socket.sendMu.Lock()
	err := socket.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	socket.sendMu.Unlock()
	if err != nil {
		log.Println("write close:", err)
	}
	socket.conn.Close()
	if socket.OnDisconnected != nil {
		socket.isConnected = false
		socket.OnDisconnected(err, *socket)
	}
}

func main() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	socket := New("ws://echo.websocket.org/", nil);
	socket.OnConnectError = func(err error, socket Socket) {
		log.Fatal("Got connect error")
	};
	socket.OnConnected = func(socket Socket) {
		log.Println("Connected to server");
	};
	socket.OnTextMessage = func(message string, socket Socket) {
		log.Println("Got message Lolwa " + message)
	};
	socket.OnPingReceived = func(data string, socket Socket) {
		log.Println("Got ping " + data)
	};
	socket.OnDisconnected = func(err error, socket Socket) {
		log.Println("Disconnected from server ")
		return
	};
	socket.Connect()

	i := 0
	for (i < 10) {
		socket.SendText("This is my sample test message")
		i++
	}

	for {
		select {
		case <-interrupt:
			log.Println("interrupt")
			socket.Close()
			return
		}
	}
}
