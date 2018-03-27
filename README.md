# GoWebsocket
Gorilla websocket based simplified client implementation in GO.

Overview
--------
This client provides following easy to implement functionality
- Support for emitting and receiving text and binary data
- Data compression
- Concurrency control
- Proxy support
- Setting request headers
- Subprotocols support
- SSL verification enable/disable

To install use

```markdown
    go get https://github.com/sacOO7/gowebsocket
```

Description
-----------

Create instance of `Websocket` by passing url of websocket-server end-point

```go
    //Create a client instance
    socket := gowebsocket.New("ws://echo.websocket.org/")
    
``` 
**Important Note** : url to websocket server must be specified with either *ws* or *wss*.

#### Registering Listeners

#### Connecting to server

#### Setting request headers

#### Setting proxy server

#### Setting compression, ssl verification and subprotocols

License
-------
Apache License, Version 2.0

