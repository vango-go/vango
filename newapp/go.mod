module newapp

go 1.23

toolchain go1.24.11

require github.com/vango-go/vango v0.1.0

require github.com/gorilla/websocket v1.5.3 // indirect

replace github.com/vango-go/vango => ../
