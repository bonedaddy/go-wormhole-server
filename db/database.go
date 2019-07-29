package db

import (
	"database/sql"
	"errors"
	"os"

	//sqlite3 driver
	_ "github.com/mattn/go-sqlite3"

	"github.com/chris-pikul/go-wormhole-server/config"
	"github.com/chris-pikul/go-wormhole-server/log"
)

var db *sql.DB

//Initialize opens the necessary database connections for SQLite3
func Initialize() error {
	if config.Opts == nil {
		panic("attempted to initialize database without a configuration loaded")
	}

	log.Info("initializing database")

	filename := config.Opts.Relay.DBFile
	if filename == "" {
		//If not running with file, then in memory should be used
		//TODO: Add in-memory replacement
		return nil
	}

	createSchema := false
	if _, err := os.Stat(filename); err != nil {
		log.Infof("creating database file %s", filename)
		createSchema = true
		os.Create(filename)
	}

	var err error
	db, err = sql.Open("sqlite3", filename)
	if err != nil {
		return err
	}
	log.Infof("database connection opened to file %s", filename)

	if createSchema {
		return CreateSchema()
	}

	//Check migration and return
	return CheckMigration()
}

//Close terminates and clears the database connection
func Close() {
	log.Info("closing database connection")
	if db != nil {
		db.Close()
	}
	db = nil
}

//Get returns the current database connection
func Get() *sql.DB {
	return db
}

//CreateSchema sets up a new database schema for use
func CreateSchema() error {
	if db == nil {
		return ErrNotOpen
	}

	log.Info("setting up database schema")
	
	_, err := db.Exec(relaySchema)
	if err != nil {
		return err
	}

	//Set the schema version
	_, err = db.Exec(`INSERT INTO version (version) VALUES ($1)`, schemaVersion)
	if err != nil {
		return err
	}

	log.Infof("set schema version to %d", schemaVersion)
	return nil
}

//CheckMigration reads the database schema version and checks
//against the current version in this binary. If they do not
//match, will attempt to migrate the schema.
func CheckMigration() error {
	if db == nil {
		return ErrNotOpen
	}

	var cur int
	row := db.QueryRow(`SELECT version FROM version`)
	if err := row.Scan(&cur); err != nil {
		if err == sql.ErrNoRows {
			//Improperly setup
			return errors.New("could not find the schema version of the database, it may be corrupt")
		}
		return err
	}

	if cur > schemaVersion {
		return errors.New("database schema version is higher then the binaries target")
	} else if cur < schemaVersion {
		log.Infof("updating db schema from %d to %d", cur, schemaVersion)
		//TODO: Implement
	}

	return nil
}

var ErrNotOpen = errors.New("database connection is not open")
