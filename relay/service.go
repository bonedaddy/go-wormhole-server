package relay

import (
	"github.com/chris-pikul/go-wormhole-server/config"
	"github.com/chris-pikul/go-wormhole-server/db"
	"github.com/chris-pikul/go-wormhole-server/log"
	"github.com/chris-pikul/go-wormhole/msg"
)

//Service encompases the actual relay service for use by
//clients. Essentially most of this package handles
//the actual network connections and message handling.
//This object is the actual implementation (or at least
//the start of it)
type Service struct {
	Welcome msg.WelcomeInfo

	Apps map[string]Application
}

//NewService initializes the relay service object
//and returns it as a pointer, if we can not start
//one then nil, error is returned instead
func NewService() (*Service, error) {
	srv := &Service{
		Apps: make(map[string]Application),
	}

	//Setup the welcome message stuff
	if config.Opts.Relay.WelcomeMOTD != "" {
		//Pointers should bind this so we can change it later?
		srv.Welcome.MOTD = &config.Opts.Relay.WelcomeMOTD
	}

	if config.Opts.Relay.WelcomeError != "" {
		//Pointers should bind this so we can change it later?
		srv.Welcome.Error = &config.Opts.Relay.WelcomeError
	}

	if config.Opts.Relay.AdvertisedVersion != "" {

		//Pointers should bind this so we can change it later?
		srv.Welcome.Version = &config.Opts.Relay.AdvertisedVersion
	}

	//Spin up the database
	err := db.Initialize()
	if err != nil {
		return nil, err
	}

	return srv, nil
}

//GetApp finds an application registered with the relay service.
//If not found, it will create and initialize the object for it
func (s *Service) GetApp(id string) *Application {
	app, ok := s.Apps[id]
	if !ok {
		//Create new application and bind it
		log.Infof("creating new application container for %s", id)
		app, _ = NewApplication(id)
		s.Apps[id] = app
	}

	return &app
}
