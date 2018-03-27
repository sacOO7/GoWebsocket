package main

import (
	"log"
	"github.com/sacOO7/gowebsocket"
	"os"
	"os/signal"
)

func main() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	socket := gowebsocket.New("ws://echo.websocket.org/");
	socket.OnConnectError = func(err error, socket gowebsocket.Socket) {
		log.Fatal("Got connect error")
	};
	socket.OnConnected = func(socket gowebsocket.Socket) {
		log.Println("Connected to server");
	};
	socket.OnTextMessage = func(message string, socket gowebsocket.Socket) {
		log.Println("Got message Lolwa " + message)
	};
	socket.OnPingReceived = func(data string, socket gowebsocket.Socket) {
		log.Println("Got ping " + data)
	};
	socket.OnDisconnected = func(err error, socket gowebsocket.Socket) {
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
