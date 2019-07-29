package transit

import (
	"bufio"
	"errors"
	"net"
	"regexp"
	"strings"

	"github.com/chris-pikul/go-wormhole-server/log"
)

//Client wraps up the net.Conn connection
//with other local properties describing
//a client connection to the transit
//service
type Client struct {
	conn net.Conn

	SentOK   bool
	GotToken bool
	TokenBuf []byte
	Token    string
	Side     string
	Mood     string

	Buddy *Client
}

//NewClient returns a new client object pointer
func NewClient(con net.Conn) *Client {
	return &Client{
		conn:     con,
		TokenBuf: make([]byte, 0),
	}
}

//Close shutsdown the client connection and
//frees any resources we may be consuming
func (c *Client) Close() {
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}

	if c.Buddy != nil && c.Buddy.conn != nil {
		c.Buddy.Close()
	}
}

//HandleConnection takes over the client connection and starts
//processing data that comes in from it
func (c *Client) HandleConnection() {
	for {
		data, err := bufio.NewReader(c.conn).ReadBytes('\n')
		if err != nil {
			if strings.Contains(err.Error(), "closed by the remote host") {
				//Ok closed by remote
				log.Info("connection closed by remote client")
				return
			}

			log.Err("failed to read from client", err)
			return
		}

		if err := c.handleData(data); err != nil {
			log.Err("failed to handle client message", err)
			return
		}
	}
}

func (c *Client) handleData(data []byte) error {
	if c.SentOK {
		//Connection was established, so start
		//pumping data back to the other client
		if c.Buddy != nil && c.Buddy.conn != nil {
			c.Buddy.conn.Write(data)
		} else {
			return errors.New("bad pipeline")
		}
		return nil
	} else if c.GotToken {
		c.conn.Write([]byte("impatient"))
		return errors.New("transit impatience failure")
	}

	c.TokenBuf = append(c.TokenBuf, data...)

	//Check the token to see if it is complete
	tokenStr := string(c.TokenBuf)
	if _, has, token := checkOldToken(tokenStr); has {
		//Old token passes
		log.Infof("accepting old version token '%s'", token)
		c.processToken(token, "")
	} else if _, has, token, side := checkNewToken(tokenStr); has {
		//New token passes
		log.Infof("accepting new token '%s' for side '%s'", token, side)
		c.processToken(token, side)
	} else {
		//Shortcut this for now
		c.conn.Write([]byte("bad handshake"))
		return errors.New("transit handshake failure")
	}

	//Keep waiting I guess
	return nil
}

var oldTokenLength = len("please relay \n") + (32 * 2)
var oldTokenMatcher = regexp.MustCompile("^please relay (\\w{64})\n")

func checkOldToken(token string) (int, bool, string) {
	//Old version token comes in form of "please relay {64}\n"
	if len(token) < oldTokenLength-1 && strings.ContainsRune(token, '\n') {
		return -1, false, ""
	}
	if len(token) < oldTokenLength {
		return 0, false, ""
	}

	matches := oldTokenMatcher.FindStringSubmatch(token)
	if matches != nil && len(matches) > 1 {
		return 1, true, matches[1]
	}

	return -1, false, ""
}

var newTokenLength = len("please relay  for side \n") + (32 * 2) + (8 * 2)
var newTokenMatcher = regexp.MustCompile("^please relay (\\w{64}) for side (\\w{16})\n")

func checkNewToken(token string) (int, bool, string, string) {
	//New version token comes in the form of "please relay {64} for side {16}\n"
	if len(token) < newTokenLength-1 && strings.ContainsRune(token, '\n') {
		return -1, false, "", "" //Will never contain a token
	} else if len(token) < newTokenLength {
		return 0, false, "", "" //Still waiting for data
	}

	matches := newTokenMatcher.FindStringSubmatch(token)
	if matches != nil && len(matches) > 1 {
		return 1, true, matches[1], matches[2]
	}

	return -1, false, "", ""
}

func (c *Client) processToken(token, side string) {
	c.Token = token
	c.Side = side
	c.Mood = "lonely"
	c.GotToken = true

	//Populate into the potentials for the service
	lock.Lock()
	defer lock.Unlock()

	if potentials, ok := pending[token]; ok { //We have potentials to search
		log.Debugf("searching %d potential connections for %s", len(potentials), token)
		var match *transitConn
		for i, ex := range potentials {
			if ex.Side == "" || side == "" || ex.Side != side {
				//We have a match for a transaction
				match = &potentials[i]

				//Swap and trim the current one
				potentials[i] = potentials[len(potentials)-1]
				potentials = potentials[:len(potentials)-1]

				for _, red := range potentials {
					if red.Client.conn != nil {
						log.Debugf("clearing out redundent in pending list %s", red.Client.conn.RemoteAddr().String())
						red.Client.conn.Write([]byte("redundent"))
					}
					red.Client.Close()
				}

				break
			}
		}

		if match != nil {
			//Dump the potentials
			delete(pending, token)

			//start connection
			match.Client.connectWith(c)
			c.connectWith(match.Client)
		}
	}

	pending[token] = []transitConn{
		transitConn{
			Side:   side,
			Client: c,
		},
	}
}

func (c *Client) connectWith(other *Client) {
	c.Mood = "happy"
	c.Buddy = other

	c.conn.Write([]byte("ok\n"))
	c.SentOK = true
}
