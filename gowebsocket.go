package main

import (
	"github.com/gorilla/websocket"
	"net/http"
	"log"
	"os"
	"os/signal"
	"errors"
)

type Socket struct {
	conn            *websocket.Conn
	url             string
	requestHeader   http.Header
	OnConnected     func(socket Socket)
	OnTextMessage   func(message string, socket Socket)
	OnBinaryMessage func(data [] byte, socket Socket)
	OnConnectError  func(err error, socket Socket)
	OnDisconnected  func(err error, socket Socket)
	OnPingReceived  func(data string, socket Socket)
	OnPongReceived  func(data string, socket Socket)
}

func New(url string, requestHeader http.Header) Socket {
	return Socket{url: url, requestHeader: requestHeader}
}

func (socket *Socket) Connect() {
	var err error;
	socket.conn, _, err = websocket.DefaultDialer.Dial(socket.url, socket.requestHeader)
	if err != nil && socket.OnConnectError != nil {
		socket.OnConnectError(err, *socket)
	}

	if socket.OnConnected != nil {
		socket.OnConnected(*socket)
	}

	socket.conn.SetPingHandler(func(appData string) error {
		if socket.OnPingReceived != nil {
			socket.OnPingReceived(appData, *socket)
		}
		return nil
	})

	socket.conn.SetPongHandler(func(appData string) error {
		if socket.OnPongReceived != nil {
			socket.OnPongReceived(appData, *socket)
		}
		return nil
	})

	socket.conn.SetCloseHandler(func(code int, text string) error {
		if socket.OnDisconnected != nil {
			socket.OnDisconnected(errors.New(text), *socket)
		}
		return nil
	})

	go func() {
		for {
			messageType, message, err := socket.conn.ReadMessage()
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
	err := socket.conn.WriteMessage(websocket.TextMessage, [] byte (message))
	if err != nil {
		log.Println("write:", err)
		return
	}
}

func (socket *Socket) SendBinary(data [] byte) {
	err := socket.conn.WriteMessage(websocket.BinaryMessage, data)
	if err != nil {
		log.Println("write:", err)
		return
	}
}

func (socket *Socket) Close() {
	err := socket.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		log.Println("write close:", err)
		if socket.OnDisconnected != nil {
			socket.OnDisconnected(err, *socket)
		}
		return
	}
	socket.conn.Close()
}

func main() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	socket := New("ws://echo.websocket.org/", nil);
	socket.OnConnected = func(socket Socket) {
		log.Println("Connected to server");
	};

	socket.OnTextMessage = func(message string, socket Socket) {
		log.Println("Got message Lolwa" + message)
	};

	socket.Connect()

	socket.SendText("This is my sample test message")

	for {
		select {
		case <-interrupt:
			log.Println("interrupt")
			socket.Close()
			return
		}
	}
}
