package transit

import (
	"context"
	"net"
	"strconv"
	"sync"

	"github.com/chris-pikul/go-wormhole-server/config"
	"github.com/chris-pikul/go-wormhole-server/log"
)

var (
	addr   string
	server net.Listener

	lock    sync.Mutex
	pending map[string][]transitConn
)

type transitConn struct {
	Side   string
	Client *Client
}

//Initialize preps the starting of the transit server.
//The transit server is a direct TCP pipeline between
//clients, this is used if all other P2P methods fail
//and an intermediary is needed after all
func Initialize() error {
	if config.Opts == nil {
		panic("attempted to initialize relay without a loaded config")
	}

	addr = net.JoinHostPort(config.Opts.Transit.Host, strconv.Itoa(int(config.Opts.Transit.Port)))

	pending = make(map[string][]transitConn, 0)

	return nil
}

//Shutdown gracefully closes the transit connections.
//Returns an error if something failed along the way.
func Shutdown(ctx context.Context) error {
	if server != nil {
		server.Close()
	}
	server = nil

	return nil
}

//Start begins the actually listening server and
//performs connections. This starts a go-routine
//within it, so this function does not block
func Start() error {
	if server != nil {
		panic("attempted to start transit server while already running")
	}

	var err error
	server, err = net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	go runTransit()

	return nil
}

func runTransit() {
	for {
		c, err := server.Accept()
		if err != nil {
			log.Err("error accepting client connection", err)
			return
		}

		go handleConnection(c)
	}
}

func handleConnection(c net.Conn) {
	log.Infof("serving tcp connection: %s", c.RemoteAddr().String())

	client := NewClient(c)
	defer client.Close()

	client.HandleConnection()
}
