package relay

import (
	"database/sql"
	"errors"
	"time"
	"sync"

	"github.com/chris-pikul/go-wormhole-server/db"
)

type MailboxListener func(MailboxMessage)
type MailboxListenerStop func()

//Mailbox holds an association with an application
//as well as its on ID. Here the client messages
//are stored for retrieval later and transmitting
//between users
type Mailbox struct {
	ID string
	AppID string

	listeners map[int]MailboxListener
	stopListeners map[int]MailboxListenerStop

	lock sync.Mutex
	listenerID int
}

//MailboxMessage is an individual entry
//within a parent Mailbox
type MailboxMessage struct {
	ID string
	AppID string
	MailboxID string
	Side string
	Phase string
	Body string
	ServerRX int64
}

type mailboxSide struct {
	mailboxID string
	opened bool
	side string
	added int64
	mood string
}

//NewMailbox returns a new mailbox address
//with the provided information
func NewMailbox(id, appID string) Mailbox {
	return Mailbox{
		ID: id,
		AppID: appID,

		listeners: make(map[int]MailboxListener),
		stopListeners: make(map[int]MailboxListenerStop),
		listenerID: 1,
	}
}

//Touch updates the db timestamp for this mailbox
func (m Mailbox) Touch() error {
	if db.Get() == nil {
		return db.ErrNotOpen
	}

	_, err := db.Get().Exec(`UPDATE mailboxes SET updated=$2 WHERE id=$1`, m.ID, time.Now().Unix())
	return err
}

//Open registers an open side on the mailbox
func (m Mailbox) Open(side string) error {
	if db.Get() == nil {
		return ErrNotOpen
	}

	msides := mailboxSide{}
	row := db.Get().QueryRow(`SELECT * FROM mailbox_sides WHERE mailbox_id=$1 AND side=$2`, m.ID, side)
	if err := row.Scan(&msides.mailboxID, &msides.opened, &msides.side, &msides.added, &msides.mood); err != nil {
		if err == sql.ErrNoRows {
			_, err = db.Get().Exec(`INSERT INTO mailbox_sides (mailbox_id, opened, side, added) 
				VALUES ($1, true, $3, $4)`, m.ID, side, time.Now().Unix())
			if err != nil {
				return err
			}
		}
	}

	return m.Touch()
}

//Close registers the mailbox as closed
func (m Mailbox) Close(side string, mood string) error {
	if db.Get() == nil {
		return db.ErrNotOpen
	}
	var err error

	//Find the mailbox object from DB
	var updated int
	var forNameplate bool
	row := db.Get().QueryRow(`SELECT * FROM mailboxes WHERE app_id=$2 AND id=$1`, m.ID, m.AppID)
	if err = row.Scan(&updated, &forNameplate); err != nil {
		if err == sql.ErrNoRows { //Bail early since the mailbox doesn't even exist
			return nil
		}
		return err
	}

	//Get the side that matches ours
	msides := mailboxSide{}
	row = db.Get().QueryRow(`SELECT * FROM mailbox_sides WHERE mailbox_id=$1 AND side=$2`, m.ID, side)
	if err := row.Scan(&msides.mailboxID, &msides.opened, &msides.side, &msides.added, &msides.mood); err != nil {
		if err == sql.ErrNoRows { //Bail early since we wheren't in this box anyways
			return nil
		}
		return err
	}

	//Clear this side
	_, err = db.Get().Exec(`UPDATE mailbox_sides SET opened=false, mood=$3 WHERE mailbox_id=$1 AND side=$2`, m.ID, side, mood)
	if err != nil {
		return err
	}

	//Check if any are open
	var opened bool
	row = db.Get().QueryRow(`SELECT COUNT(*)>0 FROM mailbox_sides WHERE mailbox_id=$1`, m.ID)
	if err = row.Scan(&opened); err != nil && err != sql.ErrNoRows {
		return err
	}

	if opened { //Leave them alone
		return nil
	}

	//None opened, start clearing it out
	return m.Delete()
}

//Delete removes the mailbox from the database
func (m Mailbox) Delete() error {
	if db.Get() == nil {
		return db.ErrNotOpen
	}
	//Probably should transaction this, but it is SQLite after all
	if _, err := db.Get().Exec(`DELETE FROM messages WHERE mailbox_id=$1`, m.ID); err != nil {
		return err
	}
	if _, err := db.Get().Exec(`DELETE FROM mailbox_sides WHERE mailbox_id=$1`, m.ID); err != nil {
		return err
	}
	if _, err := db.Get().Exec(`DELETE FROM mailboxes WHERE id=$1`, m.ID); err != nil {
		return err
	}

	m.RemoveAllListeners()

	return nil
}

//GetMessages returns the messages currently present in this mailbox
func (m Mailbox) GetMessages() ([]MailboxMessage, error) {
	var msgs []MailboxMessage

	if db.Get() == nil {
		return msgs, db.ErrNotOpen
	}

	rows, err := db.Get().Query(`SELECT * FROM messages WHERE app_id=$1 AND mailbox_id=$2
		ORDER BY server_rx ASC`, m.AppID, m.ID)
	if err != nil {
		return msgs, err
	}
	defer rows.Close()
	for rows.Next() {
		msg := MailboxMessage{}
		err = rows.Scan(&msg.ID, &msg.AppID, &msg.MailboxID, &msg.Side, &msg.Phase, &msg.Body, &msg.ServerRX)
		if err != nil {
			return msgs, err
		}

		msgs = append(msgs, msg)
	}
	err = rows.Err()
	if err != nil {
		return msgs, err
	}

	return msgs, nil
}

//AddMessage inserts a new message into the mailbox
func (m Mailbox) AddMessage(msg MailboxMessage) error {
	if db.Get() == nil {
		return db.ErrNotOpen
	}

	_, err := db.Get().Exec(`INSERT INTO messages (id, app_id, mailbox_id, side, phase, body, server_rx)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`, msg.ID, msg.AppID, msg.MailboxID, msg.Side, msg.Phase, msg.Body, msg.ServerRX)
	if err != nil {
		return err
	}

	m.broadcast(msg)

	return m.Touch()
}

//AddListener registers a callback for mailbox messages
//and returns an integer handle for de-registration
func (m *Mailbox) AddListener(listener MailboxListener, stopCallback MailboxListenerStop) int {
	m.lock.Lock()
	defer m.lock.Unlock()

	nextID := m.listenerID
	m.listenerID++

	m.listeners[nextID] = listener
	m.stopListeners[nextID] = stopCallback

	return nextID
}

//RemoveListener removes a previously registered
//listener by it's given handle
func (m *Mailbox) RemoveListener(handle int) {
	m.lock.Lock()
	defer m.lock.Unlock()

	delete(m.listeners, handle)
	delete(m.stopListeners, handle)
}

//RemoveAllListeners calls the stop callback
//on each listener, and then clears all the listners
func (m *Mailbox) RemoveAllListeners() {
	m.lock.Lock()
	defer m.lock.Unlock()

	for _, cb := range m.stopListeners {
		cb()
	}

	m.listeners = make(map[int]MailboxListener)
	m.stopListeners = make(map[int]MailboxListenerStop)
}

func (m Mailbox) broadcast(msg MailboxMessage) {
	m.lock.Lock()
	defer m.lock.Unlock()

	for _, l := range m.listeners {
		l(msg)
	}
}
