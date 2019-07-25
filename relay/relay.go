package relay

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/chris-pikul/go-wormhole-server/config"
	"github.com/chris-pikul/go-wormhole-server/log"
	"github.com/chris-pikul/go-wormhole/msg"
)

var (
	router *http.ServeMux
	server *http.Server

	clients     map[*Client]struct{}
	lockClients sync.Mutex

	register   chan *Client
	unregister chan *Client
)

//Initialize sets-up the relay servers initial systems
func Initialize() error {
	if config.Opts == nil {
		panic("attempted to initialize relay without a loaded config")
	}

	//Prepare the actual infrastructure
	clients = make(map[*Client]struct{})

	register = make(chan *Client)
	unregister = make(chan *Client)

	//Setup router
	router = http.NewServeMux()
	router.HandleFunc("/", handleIndex)
	router.HandleFunc("/v1", handleWebsocket)

	//Configure server
	server = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", config.Opts.Relay.Host, config.Opts.Relay.Port),
		Handler: router,
	}

	return nil
}

//Shutdown performs the graceful shutdown of the relay server
//using the provided context
func Shutdown(ctx context.Context) error {
	server.SetKeepAlivesEnabled(false)
	err := server.Shutdown(ctx)

	log.Info("shutdown relay server")

	return err
}

//Start spins up the relay server as a coroutine
func Start() {
	if server == nil {
		panic("attempted to start relay server that has not been initialized")
	}

	//Handle all the incoming/outgoing connections that get passed in from websocket.
	//So we run this async so it doesn't block the actual relay server
	go runRelay()

	go func() {
		log.Info("starting relay server")
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Err("closing relay server encountered an error", err)
		}
		log.Info("relay server closed")
	}()
}

func runRelay() {
	for {
		select {

		case clnt := <-register: //New client
			lockClients.Lock()
			clients[clnt] = struct{}{}
			LogInfo(clnt, "new client registered")
			lockClients.Unlock()

			clnt.sendBuffer <- msg.Welcome{
				Message: msg.NewMessage(msg.TypeWelcome),

				//TODO: add message of the day and error stuff
				Welcome: map[string]string{},
			}

		case clnt := <-unregister: //Leaving client
			lockClients.Lock()
			if _, ok := clients[clnt]; ok {
				delete(clients, clnt)
				close(clnt.sendBuffer)
			}
			LogInfo(clnt, "client unregistered")
			lockClients.Unlock()
		}
	}
}