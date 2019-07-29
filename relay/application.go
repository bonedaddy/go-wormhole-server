package relay

import (
	crand "crypto/rand"
	"database/sql"
	"encoding/base32"
	"errors"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/chris-pikul/go-wormhole-server/db"
	"github.com/chris-pikul/go-wormhole-server/log"
	"github.com/chris-pikul/go-wormhole/errs"
)

//Application holds the data for interacting with
//an individual applications usage with the relay server.
//All mailboxes are broken down into their parent apps
//so that a wider variety of client apps can exist on
//one server without conflicting with each others
//protocols
type Application struct {
	ID string

	Mailboxes map[string]*Mailbox
}

type nameplate struct {
	id        int
	appID     string
	name      string
	mailboxID string
	requestID string
}

type nameplateSide struct {
	nameplateID int
	claimed     bool
	side        string
	added       int64
}

//NewApplication creates a new application container and
//returns it as a pointer, or error if something failed.
func NewApplication(id string) (Application, error) {
	app := Application{
		ID:        id,
		Mailboxes: make(map[string]*Mailbox),
	}

	return app, nil
}

//Free is called when the service is closing and the final
//death-throws should be performed
func (a *Application) Free() {
	for _, mbox := range a.Mailboxes {
		mbox.RemoveAllListeners()
	}
	a.Mailboxes = make(map[string]*Mailbox)
}

//GetNameplateIDs returns all the nameplate IDs used
//by the current application. This should only be allowed
//if the config option AllowList is true.
func (a Application) GetNameplateIDs() ([]string, error) {
	res := make([]string, 0)

	if db.Get() == nil {
		return res, db.ErrNotOpen
	}

	rows, err := db.Get().Query(`SELECT DISTINCT name FROM nameplates WHERE app_id=$1`, a.ID)
	if err != nil {
		log.Err("failed to get nameplate IDs from DB", err)
		return res, err
	}

	defer rows.Close()
	for rows.Next() {
		var name string
		if err = rows.Scan(&name); err != nil {
			log.Err("scanning row for nameplate name", err)
			return res, err
		}
		res = append(res, name)
	}
	if err = rows.Err(); err != nil {
		log.Err("scanning rows for nameplates", err)
		return res, err
	}

	return res, nil
}

//FindNameplate attempts to find an available nameplate
//to return back for clients to use
func (a Application) FindNameplate() (string, error) {
	claimed, err := a.GetNameplateIDs()
	if err != nil {
		log.Err("getting claimed nameplates for FindNameplate", err)
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
		id := strconv.Itoa(randRange(1000, 1000^1000))

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

	log.Err("no available nameplates")
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
				log.Err("could not add mailbox for ClaimNameplate", err)
				return "", err
			}

			npres, e := db.Get().Exec(`INSERT INTO nameplates (app_id, name, mailbox_id) 
				VALUES ($1, $2, $3)`, a.ID, name, mbid)
			if e != nil {
				log.Err("could not create nameplate for ClaimNameplate", err)
				return "", e
			}
			npidl, e := npres.LastInsertId()
			if e != nil {
				log.Err("failed to get new nameplate id for ClaimNameplate", err)
				return "", e
			}
			npid = int(npidl)
		} else {
			log.Err("failed to find existing nameplates for ClaimNameplate", err)
			return "", err
		}
	} else {
		npid = np.id
		mbid = np.mailboxID
	}

	nps := nameplateSide{}
	row = db.Get().QueryRow(`SELECT * FROM nameplate_sides WHERE nameplate_id=$1 AND side=$2`, npid, side)
	if err := row.Scan(&nps.nameplateID, &nps.claimed, &nps.side, &nps.added); err != nil {
		if err == sql.ErrNoRows {
			_, err = db.Get().Exec(`INSERT INTO nameplate_sides (nameplate_id, claimed, side, added)
				VALUES ($1, true, $2, $3)`, npid, side, time.Now().Unix())
			if err != nil {
				log.Err("inserting new nameplate side for ClaimNameplate", err)
				return "", err
			}
		} else {
			log.Err("selecting existing nameplate sides for ClaimNameplate", err)
			return "", err
		}
	}

	if nps.claimed {
		return "", errs.ErrReclaimNameplate //Cannot reclaim from the same side
	}

	_, err := a.OpenMailbox(mbid, side)
	if err != nil {
		log.Err("could not open mailbox for ClaimNameplate", err)
		return "", err
	}

	var sidesOpen int
	row = db.Get().QueryRow(`SELECT COUNT(*) FROM nameplate_sides WHERE nameplate_id=$1`, npid)
	if err := row.Scan(&sidesOpen); err != nil && err != sql.ErrNoRows {
		log.Err("counting open nameplate_sides for ClaimNameplate", err)
		return "", err
	}

	if sidesOpen > 2 {
		log.Warnf("nameplate %s is crowded", npid)
		return "", errs.ErrNameplateCrowded
	}

	return mbid, nil
}

//AllocateNameplate generates a new nameplate ID and associates
//with a mailbox. The returned value is the nameplate ID
func (a Application) AllocateNameplate(side string) (string, error) {
	nameplate, err := a.FindNameplate()
	if err != nil {
		log.Err("could not find nameplate for AllocateNameplate", err)
		return "", err
	}

	_, err = a.ClaimNameplate(nameplate, side)
	if err != nil {
		log.Err("could not claim nameplate for AllocateNameplate", err)
		return "", err
	}

	return nameplate, nil
}

//ReleaseNameplate removes the claim on a nameplates side.
//If no other claims are on the nameplate, then the whole thing is cleared out
func (a Application) ReleaseNameplate(name, side string) error {
	if db.Get() == nil {
		return db.ErrNotOpen
	}

	//Check that the nameplate exists
	np := nameplate{}
	row := db.Get().QueryRow(`SELECT * FROM nameplates WHERE app_id=$1 AND name=$2`, a.ID, name)
	if err := row.Scan(&np.id, &np.appID, &np.name, &np.mailboxID, &np.requestID); err != nil {
		if err == sql.ErrNoRows {
			return nil //Nothing to do, no nameplate found
		}
		log.Err("getting existing nameplates for ReleaseNameplate", err)
		return err
	}

	//Check that the side exists
	nps := nameplateSide{}
	row = db.Get().QueryRow(`SELECT * FROM nameplate_sides WHERE nameplate_id=$1 AND side=$2`, np.id, side)
	if err := row.Scan(&nps.nameplateID, &nps.claimed, &nps.side, &nps.added); err != nil {
		if err == sql.ErrNoRows {
			return nil //Notihing to do, no claimed sides
		}
		log.Err("getting nameplate sides for ReleaseNameplate", err)
		return err
	}

	//Unclaim the side
	_, err := db.Get().Exec(`UPDATE nameplate_sides SET claimed=false WHERE nameplate_id=$1 AND side=$2`, np.id, side)
	if err != nil {
		log.Err("updating nameplate sides for ReleaseNameplate", err)
		return err
	}

	//Check if any remaining claims
	var rem int
	row = db.Get().QueryRow(`SELECT COUNT(*) FROM nameplate_sides WHERE nameplate_id=$1 AND claimed=true`, np.id)
	if err := row.Scan(&rem); err != nil && err != sql.ErrNoRows {
		log.Err("counting nameplate sides for ReleaseNameplate", err)
		return err
	}

	if rem > 0 {
		return nil //Still active claims
	}

	//Delete the nameplate and free it
	_, err = db.Get().Exec(`DELETE FROM nameplate_sides WHERE nameplate_id=$1`, np.id)
	if err != nil {
		log.Err("deleting nameplate sides for ReleaseNameplate", err)
		return err
	}

	_, err = db.Get().Exec(`DELETE FROM nameplates WHERE id=$1`, np.id)
	if err != nil {
		log.Err("deleting nameplate for ReleaseNameplate", err)
	}
	return err
}

//AddMailbox creates a new mailbox in the application
func (a Application) AddMailbox(id string, forNameplate bool, side string) error {
	if db.Get() == nil {
		return db.ErrNotOpen
	}

	var exists bool
	row := db.Get().QueryRow(`SELECT COUNT(*)>0 FROM mailboxes WHERE app_id=$1 AND id=$2`, a.ID, id)
	if err := row.Scan(&exists); err != nil && err != sql.ErrNoRows {
		log.Err("getting mailboxes for AddMailbox", err)
		return err
	}

	if exists {
		return nil
	}

	_, err := db.Get().Exec(`INSERT INTO mailboxes (app_id, id, for_nameplate, updated)
		VALUES ($1, $2, $3, $4)`, a.ID, id, forNameplate, time.Now().Unix())
	if err != nil {
		log.Err("inserting new mailbox for AddMailbox", err)
	}
	return err
}

//OpenMailbox marks the mailbox as opened
func (a Application) OpenMailbox(id, side string) (*Mailbox, error) {
	if db.Get() == nil {
		return nil, db.ErrNotOpen
	}

	//Ensure existance
	err := a.AddMailbox(id, false, side)
	if err != nil {
		log.Err("adding mailbox for OpenMailbox", err)
		return nil, err
	}

	mbox, has := a.Mailboxes[id]
	if !has {
		mbox = NewMailbox(id, a.ID)
		a.Mailboxes[id] = mbox
	}

	err = mbox.Open(side)
	if err != nil {
		log.Err("opening mailbox for OpenMailbox", err)
		return nil, err
	}

	var sidesOpen int
	row := db.Get().QueryRow(`SELECT COUNT(*) FROM mailbox_sides WHERE mailbox_id=$1`, id)
	if err := row.Scan(&sidesOpen); err != nil && err != sql.ErrNoRows {
		log.Err("counting mailbox sides for OpenMailbox", err)
		return nil, err
	}

	if sidesOpen > 2 {
		return nil, errs.ErrMailboxCrowded
	}

	return mbox, nil
}

//FreeMailbox removes a mailbox listing from the application memory.
//Does not remove it from the database
func (a Application) FreeMailbox(id string) {
	_, ok := a.Mailboxes[id]
	if ok {
		delete(a.Mailboxes, id)
	}
}

//Cleanup updates and removes mailboxes and nameplates as
//needed via timeouts.
func (a *Application) Cleanup(since int64) error {
	if db.Get() == nil {
		return db.ErrNotOpen
	}
	log.Infof("cleaning up application %s", a.ID)

	//Touch boxes if someone is listening
	//^ great comment, I know
	for _, mbox := range a.Mailboxes {
		if mbox.HasListeners() {
			log.Infof("touching %s because of listeners", mbox.ID)
			mbox.Touch()
		}
	}

	//Prep to clean old mailboxes
	oldMboxes := make([]string, 0)
	{ //Scope for the defer
		rows, err := db.Get().Query("SELECT * FROM mailboxes WHERE app_id=$1", a.ID)
		if err != nil && err != sql.ErrNoRows {
			log.Err("getting mailboxes for application Cleanup", err)
			return err
		}
		defer rows.Close()
		for rows.Next() {
			mbox := mailboxRaw{}
			if err := rows.Scan(&mbox.id, &mbox.appID, &mbox.updated, &mbox.forNameplate); err != nil {
				log.Err("scanning mailbox for application Cleanup", err)
				return err
			}

			if mbox.updated <= since {
				oldMboxes = append(oldMboxes, mbox.id)
			}
		}
		if err := rows.Err(); err != nil {
			log.Err("scanning mailbox rows for application Cleanup", err)
			return err
		}
	}

	//Prep to clean old nameplates
	oldNameplates := make([]int, 0)
	{ //Scope for the defer
		rows, err := db.Get().Query(`SELECT * FROM nameplates WHERE app_id=$1`, a.ID)
		if err != nil && err != sql.ErrNoRows {
			log.Err("selecting nameplates for application Cleanup", err)
			return err
		}
		defer rows.Close()
		for rows.Next() {
			np := nameplate{}
			if err := rows.Scan(&np.id, &np.appID, &np.name, &np.mailboxID, &np.requestID); err != nil {
				log.Err("scanning nameplate for application Cleanup", err)
				return err
			}

			found := false
			for _, mid := range oldMboxes {
				if mid == np.mailboxID {
					found = true
					break
				}
			}

			if found {
				oldNameplates = append(oldNameplates, np.id)
			}
		}
		if err := rows.Err(); err != nil {
			log.Err("scanning nameplate rows for application Cleanup", err)
			return err
		}
	}

	//Clear out old nameplates
	for _, np := range oldNameplates {
		if _, err := db.Get().Exec(`DELETE FROM nameplate_sides WHERE nameplate_id=$1`, np); err != nil {
			log.Err("deleting nameplate sides for application Cleanup", err)
			return err
		}

		if _, err := db.Get().Exec(`DELETE FROM nameplates WHERE id=$1`, np); err != nil {
			log.Err("deleting nameplates for application Cleanup", err)
			return err
		}

		log.Infof("cleaned nameplate %d", np)
	}

	//Clear out old mailboxes
	for _, mbid := range oldMboxes {
		if _, err := db.Get().Exec(`DELETE FROM messages WHERE mailbox_id=$1`, mbid); err != nil {
			log.Err("deleting messages for application Cleanup", err)
			return err
		}

		if _, err := db.Get().Exec(`DELETE FROM mailbox_sides WHERE mailbox_id=$1`, mbid); err != nil {
			log.Err("deleting mailbox sides for application Cleanup", err)
			return err
		}

		if _, err := db.Get().Exec(`DELETE FROM mailboxes WHERE id=$1`, mbid); err != nil {
			log.Err("deleting mailbox for application Cleanup", err)
			return err
		}

		log.Infof("cleaned mailbox %s", mbid)
	}

	return nil
}

//StillInUse returns true if the application (by ID) is still
//being used, or registered, in the database. If it is not,
//then it's safe to delete it during cleaning
func (a Application) StillInUse() bool {
	if db.Get() == nil {
		return false
	}

	var inUse bool
	row := db.Get().QueryRow(`SELECT COUNT(*)>0 FROM mailboxes WHERE app_id=$1`, a.ID)
	row.Scan(&inUse)
	if inUse {
		return true
	}

	row = db.Get().QueryRow(`SELECT COUNT(*)>0 FROM nameplates WHERE app_id=$1`, a.ID)
	row.Scan(&inUse)
	if inUse {
		return true
	}

	return false
}

func randRange(l, h int) int {
	return rand.Intn(h-l) + l
}

func generateMailboxID() string {
	b := make([]byte, 8)
	crand.Read(b)

	id := base32.StdEncoding.EncodeToString(b)
	id = strings.ReplaceAll(id, "=", "")
	id = strings.ToLower(id)

	return id
}
