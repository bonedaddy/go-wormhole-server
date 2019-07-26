# Magic Wormhole Server - WIP

Safely transfer files, or text, from one device to another (peer-to-peer) without the need for third-party accounts or custom servers. Clients can transfer media between devices (directly if possible) using encryption based on mutually shared passwords. The passwords are called "wormhole codes", and are simple phrases made up of two or more words (and a small channel number). Each client, or side of the connection, must enter the same code in order to safely establish a communication channel. Magic Wormhole does not facilitate this code sharing, so it's up to the sending client to notify the receiver of the code. Because of this fact, the security of the transfer can be of the upmost quality.

## About This Package

In order to get a working P2P connection working, a rendezvous server is required to help traverse the unknowns of the internet (IP address discovery, NAT, etc.). This package intends to faithfully support the [Magic Wormhole Protocol](https://github.com/warner/go-wormhole) for relaying connection messages, and if needed a data pipeline between two clients. Unlike the original library, I chose to combine the relay (rendezvous) and transit servers together.

The relay service is implemented using WebSockets to answer client commands. It effectively is just a message queue / publish-subscribe setup using "mailboxes". The relay server creates and binds "nameplates" which are the channel ID for the connection. This nameplate is used in the "wormhole code" that the peers are required to communicate to each other.

## Installation

go-wormhole is made using Go, as such, all the platforms Go supports should be acceptable for go-wormhole as well. Binaries will be provided as I make releases, but "from source" is always an option if you have a Go installation ready.

#### From Source

Requires Go toolkit version 1.12.5 or higher. Due to the dependency on [go-sqlite3](https://github.com/mattn/go-sqlite3) `gcc` is also required, most likely with the `CGO_ENABLED` environment variable as well.

1. `go get github.com/chris-pikul/go-wormhole-server`
2. `go install github.com/chris-pikul/go-wormhole-server`

## Usage

Being a CLI application, using `go-wormhole-server help` will display the help and usage message.

```
COMMANDS:
     serve    serve both relay, and transit requests (default command)
     clean    clears the SQLite database file
     relay    run as relay server (rendezvous) only
     transit  run as transit server (piping) only
     help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --config value, -c value       configuration JSON file to use instead of options (empty = no config)
   --relay-host value             host address or IP for the listening interface
   --relay-port value             port number to listen on (default: 4000)
   --transit-host value           host address or IP for the listening interface
   --transit-port value           port number to listen on (default: 4001)
   --db value, -d value           path to SQLite database file (default: "wormhole-relay.db")
   --no-list                      disable the 'list' request
   --advert-version value         version to recommend to clients
   --cleaning value, -C value     time interval inbetween cleaning channels in minutes (default: 5)
   --channel-exp value, -e value  channel expiration time in minutes (should be larger then cleaning period) (default: 11)
   --log value, -l value          file to write usage/error logs to (empty does not write logs)
   --log-level value, -L value    logging level to use options are [DEBUG|INFO|WARN|ERROR] (default: "INFO")
   --log-blur value               round out access times to seconds provided in logging to improve privacy (default: 1)
   --help, -h                     show help
   --version, -v                  print the version
```

CLI flags are a bit annoying at times, so they can all be ignored using the `--config` option providing a JSON configuration file. 

## License & Basis

This codebase, written by Chris Pikul, is licensed under MIT License, see LICENSE for more details.

The original library that this package is based on is also MIT licensed. Original libraries:

- [github.com/warner/magic-wormhole](https://github.com/warner/magic-wormhole)
- [github.com/warner/magic-wormhole-mailbox-server](https://github.com/warner/magic-wormhole-mailbox-server)
- [github.com/warner/magic-wormhole-transit-relay](https://github.com/warner/magic-wormhole-transit-relay)