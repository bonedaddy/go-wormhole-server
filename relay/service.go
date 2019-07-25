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

//GetAllApps returns all the application IDs in memory, and in
//the database.
//I will admit, this function was A LOT shorter in the Python
//version.
func (s Service) GetAllApps() ([]string, error) {
	if db.Get() == nil {
		return []string{}, db.ErrNotOpen
	}

	apps := make([]string, 0)

	{ //Scope for the defer
		rows, err := db.Get().Query(`SELECT DISTINCT app_id FROM nameplates`)
		if err != nil {
			return apps, err
		}
		defer rows.Close()
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				return apps, err
			}

			found := false
			for _, aID := range apps {
				if aID == id {
					found = true
					break
				}
			}

			if !found {
				apps = append(apps, id)
			}
		}
		if err := rows.Err(); err != nil {
			return apps, err
		}
	}

	{ //Scope for the defer
		rows, err := db.Get().Query(`SELECT DISTINCT app_id FROM mailboxes`)
		if err != nil {
			return apps, err
		}
		defer rows.Close()
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				return apps, err
			}

			found := false
			for _, aID := range apps {
				if aID == id {
					found = true
					break
				}
			}

			if !found {
				apps = append(apps, id)
			}
		}
		if err := rows.Err(); err != nil {
			return apps, err
		}
	}

	{ //Scope for the defer
		rows, err := db.Get().Query(`SELECT DISTINCT app_id FROM messages`)
		if err != nil {
			return apps, err
		}
		defer rows.Close()
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				return apps, err
			}

			found := false
			for _, aID := range apps {
				if aID == id {
					found = true
					break
				}
			}

			if !found {
				apps = append(apps, id)
			}
		}
		if err := rows.Err(); err != nil {
			return apps, err
		}
	}

	return apps, nil
}

//CleanApps iterates the apps registered to the service
//and runs the cleaining process on each one
func (s *Service) CleanApps(since int64) error {
	log.Info("cleaning all applications")

	apps, err := s.GetAllApps()
	if err != nil {
		return err
	}

	deadApps := make([]string, 0)
	for _, appID := range apps {
		app, ok := s.Apps[appID]
		if ok {
			err := app.Cleanup(since)
			if err != nil {
				return err
			}

			if !app.StillInUse() {
				//OK to clear this one
				deadApps = append(deadApps, appID)
			}
		}
	}

	//No longer in use, dump the memory
	for _, appID := range deadApps {
		delete(s.Apps, appID)
	}

	log.Info("completed cleaning")
	return nil
}