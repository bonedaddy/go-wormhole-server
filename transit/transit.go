package transit

import "context"

//Initialize preps the starting of the transit server.
//The transit server is a direct TCP pipeline between
//clients, this is used if all other P2P methods fail
//and an intermediary is needed after all
func Initialize() error {
	return nil
}

//Shutdown gracefully closes the transit connections.
//Returns an error if something failed along the way.
func Shutdown(ctx context.Context) error {
	return nil
}

//Start begins the actually listening server and
//performs connections. This starts a go-routine
//within it, so this function does not block
func Start() {

}
