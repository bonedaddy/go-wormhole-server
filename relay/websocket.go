package relay

import (
	"net/http"
	"time"

	"github.com/chris-pikul/go-wormhole-server/log"
	"github.com/chris-pikul/go-wormhole/msg"
	"github.com/gorilla/websocket"
)

var upgrader websocket.Upgrader

func initWebsocket() error {
	upgrader = websocket.Upgrader{
		HandshakeTimeout: time.Minute,

		ReadBufferSize:    4096,
		WriteBufferSize:   4096,
		EnableCompression: true,

		Subprotocols: []string{},
	}

	return nil
}

func handleWebsocket(w http.ResponseWriter, r *http.Request) {
	respHeader := http.Header{}

	conn, err := upgrader.Upgrade(w, r, respHeader)
	if err != nil {
		log.Warnf("upgrading connection to websocket failed: %s", err.Error())
		return
	}

	client := &Client{
		conn:       conn,
		sendBuffer: make(chan msg.IMessage, 64),
	}
	register <- client

	go client.watchWrites()
	go client.watchReads()
}
