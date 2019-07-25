package relay

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/chris-pikul/go-wormhole-server/log"
	"github.com/chris-pikul/go-wormhole/errs"
	"github.com/chris-pikul/go-wormhole/msg"
	"github.com/gorilla/websocket"
)

const (
	readWait  = 60 * time.Second
	writeWait = 10 * time.Second

	pingInterval = (readWait * 9) / 10

	maxMessageSize = 1024
)

//Client wraps up the websocket connection
//with a sending buffer and functions for transfering messages
type Client struct {
	conn       *websocket.Conn
	sendBuffer chan msg.IMessage

	App  string
	Side string
}

//IsBound returns true if the client has already bound to the server
func (c Client) IsBound() bool {
	return c.App != "" && c.Side != ""
}

func (c *Client) watchReads() {
	defer func() {
		unregister <- c
		c.conn.Close() //Close the actual connection here
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(readWait))

	//Setup the ping/pong response outside of message processing
	//which basically just extends the connection life
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(readWait))
		LogDebug(c, "received pong from client")
		return nil
	})

	//Start accepting messages and processing them
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil { // Read/Connection error
			if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				LogErr(c, "reading from socket connection", err)
			}
			break //Leave the loop, so unregister
		}

		LogDebugf(c, "received message from client %s", string(message))

		//Process the message
		c.OnMessage(message)
	}
}

func (c *Client) watchWrites() {
	ticker := time.NewTicker(pingInterval)
	defer func() {
		ticker.Stop()
		c.conn.Close() //Double check the connection is closed
	}()

	for {
		select {
		case msgObj, ok := <-c.sendBuffer: //Read messages to send
			//Give them 10 seconds to take the new message
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))

			if !ok {
				//Channel was closed
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				log.Debug("write channel was closed, disconnecting client")
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil { //Failed to get a write channel
				log.Debug("failed to get a writer for client")
				return
			}
			if err = json.NewEncoder(w).Encode(msgObj); err != nil {
				LogErr(c, "failed to encode message", err)
			}

			if err := w.Close(); err != nil { //Writer failure
				return
			}
		case <-ticker.C: //Ping check for keeping the connection alive
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Debug("failed to write ping, disconnecting client")
				return //Failed to write ping
			}
			LogDebug(c, "sent ping message to client")
		}
	}
}

//OnConnect is called when the client has successfully been registered
//to the server
func (c *Client) OnConnect() {
	c.sendBuffer <- msg.Welcome{
		Message: msg.NewServerMessage(msg.TypeWelcome),

		Info: service.Welcome,
	}
}

//OnMessage called when a message from the client is received
//and it needs to be handled/processed.
//From this point we have a message as only the bytes and
//need to handle it appropriately. We need to do all steps
//and validation here as necessary. So this will be broken
//down into the message types.
func (c *Client) OnMessage(src []byte) {
	mt, im, err := msg.ParseClient(src)
	if err != nil {
		c.messageError(err, src)
		return
	}

	//TODO: Find the ID thing
	c.sendBuffer <- msg.Ack{
		Message: msg.NewServerMessage(msg.TypeAck),
		ID:      0, //Not sure where this comes from yet
	}

	//Quit ahead if we haven't bound and aren't going to
	if c.IsBound() == false && mt != msg.TypePing && mt != msg.TypeBind {
		c.messageError(errs.ErrBindFirst, src)
		return
	}

	switch mt {
	case msg.TypePing:
		m := im.(msg.Ping)
		c.HandlePing(m)
	default:
		c.messageError(fmt.Errorf("unsuported command '%s'", mt.String()), src)
	}
}

//when bad or malformed messages appear, this method
//will create the necessary error response and send it to
//the client. These are generally only validation/protocol
//errors and not actual networking errors
func (c *Client) messageError(err error, orig []byte) {
	if err == msg.ErrUnknown {
		//Convert to the reply type
		err = errs.ErrUnknownType
	}
}

//HandlePing handles ping messages and responds back
//with the matching Pong message
func (c *Client) HandlePing(m msg.Ping) {
	c.sendBuffer <- msg.Pong{
		Message: msg.NewServerMessage(msg.TypePong),

		Pong: m.Ping,
	}
	LogDebugf(c, "received ping %d", m.Ping)
}

//HandleBind handles bind messages.
func (c *Client) HandleBind(m msg.Bind) error {
	if c.IsBound() {
		return errs.ErrBound //Already bound
	} else if c.App == "" {
		return errs.ErrBindAppID
	} else if c.Side == "" {
		return errs.ErrBindSide
	}

	//TODO: Left off here

	return nil
}
