package relay

import (
	"errors"
	"encoding/base32"
	"database/sql"
	"math"
	"math/rand"
	crand "crypto/rand"
	"strconv"
	"strings"
	"time"

	"github.com/chris-pikul/go-wormhole-server/db"
	"github.com/chris-pikul/go-wormhole-server/log"
)

//Application holds the data for interacting with
//an individual applications usage with the relay server.
//All mailboxes are broken down into their parent apps
//so that a wider variety of client apps can exist on
//one server without conflicting with each others
//protocols
type Application struct {
	ID string

	Mailboxes map[string]Mailbox
}

type nameplate struct {
	id int
	appID string
	name string
	mailboxID string
	requestID string
}

type nameplateSide struct {
	nameplateID int
	claimed bool
	side string
	added int64
}

//NewApplication creates a new application container and
//returns it as a pointer, or error if something failed.
func NewApplication(id string) (Application, error) {
	app := Application{
		ID: id,
		Mailboxes: make(map[string]Mailbox),
	}

	return app, nil
}

//GetNameplateIDs returns all the nameplate IDs used
//by the current application. This should only be allowed
//if the config option AllowList is true.
func (a Application) GetNameplateIDs() ([]string, error) {
	res := make([]string, 0)

	if db.Get() == nil {
		return res, errors.New("database connection is not open")
	}

	rows, err := db.Get().Query(`SELECT DISTINCT name FROM nameplates WHERE app_id=$1`, a.ID)
	if err != nil {
		return res, err
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err = rows.Scan(&name); err != nil {
			return res, err
		}
		res = append(res, name)
	}
	if err = rows.Err(); err != nil {
		return res, err
	}

	return res, nil
}

//FindNameplate attempts to find an available nameplate
//to return back for clients to use
func (a Application) FindNameplate() (string, error) {
	claimed, err := a.GetNameplateIDs()
	if err != nil {
		return "", err
	}

	//Attempt to generate a pseudo random nameplate ID
	for i := 1; i < 4; i++ {
		avail := make([]string, 0)

		low := int(math.Pow(10, float64(i-1)))
		high := int(math.Pow(10, float64(i)))
		for j := low; j < high; j++ {
			id := strconv.Itoa(j)
			taken := false
			for _, c := range claimed {
				if c == id {
					taken = true
					break
				}
			}

			if !taken {
				avail = append(avail, id)
			}
		}

		if len(avail) > 0 {
			return avail[randRange(0, len(avail))], nil
		}
	}

	//Too many collisions, try something gigantic
	for i := 0; i < 1000; i++ {
		id := strconv.Itoa( randRange(1000, 1000^1000) )

		taken := false
		for _, c := range claimed {
			if c == id {
				taken = true
				break
			}
		}

		if !taken {
			return id, nil
		}
	}

	return "", errors.New("no available nameplate IDs")
}

//ClaimNameplate claims a nameplate and it's respective mailbox.
//Returns the mailbox ID, or an error if one occured
func (a Application) ClaimNameplate(name, side string) (string, error) {
	if db.Get() == nil {
		return "", db.ErrNotOpen
	}

	var mbid string
	var npid int
	np := nameplate{}
	row := db.Get().QueryRow(`SELECT * FROM nameplates WHERE app_id=$1 AND name=$2`, a.ID, name)
	if err := row.Scan(&np.id, &np.appID, &np.name, &np.mailboxID, &np.requestID); err != nil {
		if err == sql.ErrNoRows {
			log.Infof("creating nameplate %s for application %s", name, a.ID)

			mbid = generateMailboxID()
			err = a.AddMailbox(mbid, true, side)
			if err != nil {
				return "", err
			}

			npres, e := db.Get().Exec(`INSERT INTO nameplates (app_id, name, mailbox_id) 
				VALUES ($1, $2, $3)`, a.ID, name, mbid)
			if e != nil {
				return "", e
			}
			npidl, e := npres.LastInsertId()
			if e != nil {
				return "", e
			}
			npid = int(npidl)
		}
	} else {
		npid = np.id
		mbid = np.mailboxID
	}

	nps := nameplateSide{}
	row = db.Get().QueryRow(`SELECT * FROM nameplate_sides WHERE nameplate_id=$1 AND side=$2`, npid, side)
	if err := row.Scan(&nps.nameplateID, &nps.claimed, &nps.side, &nps.added); err != nil {
		if err == sql.ErrNoRows {
			_, err = db.Get().Exec(`INSERT INTO nameplate_sides (nameplates_id, claimed, side, added)
				VALUES ($1, true, $2, $3)`, npid, side, time.Now().Unix())
			if err != nil {
				return "", err
			}
		} else {
			return "", err
		}
	}

	if nps.claimed {
		return "", nil //Cannot reclaim from the same side
	}

	err := a.OpenMailbox(mbid, side)
	if err != nil {
		return "", err
	}

	var sidesOpen int
	row = db.Get().QueryRow(`SELECT COUNT(*) FROM nameplate_sides WHERE nameplate_id=$1`)
	if err := row.Scan(&sidesOpen); err != nil && err != sql.ErrNoRows {
		return "", errors.New("nameplate crowded")
	}

	return mbid, nil
}

//AllocateNameplate generates a new nameplate ID and associates
//with a mailbox. The returned value is the nameplate ID
func (a Application) AllocateNameplate(side string) (string, error) {
	nameplate, err := a.FindNameplate()
	if err != nil {
		return "", err
	}

	_, err = a.ClaimNameplate(nameplate, side)
	if err != nil {
		return "", err
	}

	return nameplate, nil
}

//AddMailbox creates a new mailbox in the application
func (a Application) AddMailbox(id string, forNameplate bool, side string) error {
	if db.Get() == nil {
		return db.ErrNotOpen
	}

	var exists bool
	row := db.Get().QueryRow(`SELECT * FROM mailboxes WHERE app_id=$1 AND id=$2`, a.ID, id)
	if err := row.Scan(&exists); err != nil && err != sql.ErrNoRows {
		return err
	}

	if exists {
		return nil
	}

	_, err := db.Get().Exec(`INSERT INTO mailboxes (app_id, id, for_nameplate, updated)
		VALUES ($1, $2, $3, $4`, a.ID, id, forNameplate, time.Now().Unix())
	return err
}

//OpenMailbox marks the mailbox as opened
func (a Application) OpenMailbox(id, side string) error {
	if db.Get() == nil {
		return db.ErrNotOpen
	}

	//Ensure existance
	err := a.AddMailbox(id, false, side)
	if err != nil {
		return err
	}

	mbox, has := a.Mailboxes[id]
	if !has {
		mbox = NewMailbox(id, a.ID)
		a.Mailboxes[id] = mbox
	}

	err = mbox.Open(side)
	if err != nil {
		return err
	}

	var sidesOpen int
	row := db.Get().QueryRow(`SELECT COUNT(*) FROM mailbox_sides WHERE mailbox_id=$1`, id)
	if err := row.Scan(&sidesOpen); err != nil && err != sql.ErrNoRows {
		return err
	}

	if sidesOpen > 2 {
		return errors.New("mailbox crowded")
	}

	return nil
}

func randRange(l, h int) int {
	return rand.Intn(h - l) + l
}

func generateMailboxID() string {
	b := make([]byte, 8)
	crand.Read(b)

	id := base32.StdEncoding.EncodeToString(b)
	id = strings.ReplaceAll(id, "=", "")
	id = strings.ToLower(id)

	return id
}