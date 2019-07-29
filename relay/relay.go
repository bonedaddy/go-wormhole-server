package relay

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/chris-pikul/go-wormhole-server/config"
	"github.com/chris-pikul/go-wormhole-server/db"
	"github.com/chris-pikul/go-wormhole-server/log"
)

var (
	router  *http.ServeMux
	server  *http.Server
	service *Service

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

	var err error

	//Spin up the service, without it we should fail
	service, err = NewService()
	if err != nil {
		return err //Pass it up to the CLI
	}

	//Prepare the connection infrastructure
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
	var err error

	if server != nil {
		server.SetKeepAlivesEnabled(false)
		err = server.Shutdown(ctx)
		log.Info("shutdown relay server")
	}

	db.Close()

	log.Info("completed shutdown")
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

	//Allow the cleaning process to run
	go runCleaning()
}

//CleanNowPure runs the cleaning operation without actually spinning up the
//service resources
func CleanNowPure() error {
	if config.Opts == nil {
		panic("attempted to initialize relay without a loaded config")
	}
	var err error

	//Spin up the service, without it we should fail
	service, err = NewService()
	if err != nil {
		return err //Pass it up to the CLI
	}

	err = service.CleanApps(time.Now().Unix())
	if err != nil {
		return err
	}

	//Shutdown manually
	db.Close()

	return nil
}

func runRelay() {
	for {
		select {

		case clnt := <-register: //New client
			lockClients.Lock()
			clients[clnt] = struct{}{}
			LogInfo(clnt, "new client registered")
			lockClients.Unlock()

			clnt.OnConnect()

		case clnt := <-unregister: //Leaving client
			lockClients.Lock()
			if _, ok := clients[clnt]; ok {
				clnt.Close()
				delete(clients, clnt)
			}
			LogInfo(clnt, "client unregistered")
			lockClients.Unlock()
		}
	}
}

func runCleaning() {
	if config.Opts == nil {
		return //No options available
	}

	if config.Opts.Relay.CleaningInterval == 0 {
		log.Warn("cleaning interval was too small! Check configuration")
		return
	}

	dur := time.Minute * time.Duration(config.Opts.Relay.CleaningInterval)

	ticker := time.NewTicker(dur)
	lastCleaning := time.Now().Add(-dur) //simulate the time to be before now so we don't over clean the first time
	for t := range ticker.C {
		if service != nil {
			err := service.CleanApps(lastCleaning.Unix())
			if err != nil {
				log.Err("failed to clean relay server", err)
			}
		}

		lastCleaning = t
	}
}
