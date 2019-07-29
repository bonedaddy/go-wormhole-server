package db

const schemaVersion = 1

const relaySchema = `
CREATE TABLE version (
	version INTEGER NOT NULL
);

-- Relay data

CREATE TABLE mailboxes (
	id VARCHAR PRIMARY KEY,
	app_id VARCHAR,
	updated INTEGER,
	for_nameplate BOOLEAN
);
CREATE INDEX idx_mailboxes ON mailboxes (app_id, id);

CREATE TABLE mailbox_sides (
	mailbox_id VARCHAR REFERENCES mailboxes(id),
	opened BOOLEAN,
	side VARCHAR,
	added INTEGER,
	mood VARCHAR
);
CREATE INDEX idx_mailbox_sides ON mailbox_sides (mailbox_id);

CREATE TABLE messages (
	id VARCHAR,
	app_id VARCHAR,
	mailbox_id VARCHAR REFERENCES mailboxes(id),
	side VARCHAR,
	phase VARCHAR,
	body VARCHAR,
	server_rx INTEGER
);
CREATE INDEX idx_messages ON messages (app_id, mailbox_id);

CREATE TABLE nameplates (
	id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
	app_id VARCHAR,
	name VARCHAR,
	mailbox_id VARCHAR REFERENCES mailboxes(id),
	request_id VARCHAR DEFAULT ''
);
CREATE INDEX idx_nameplates ON nameplates (app_id, name);
CREATE INDEX idx_nameplates_mailbox ON nameplates (app_id, mailbox_id);
CREATE INDEX idx_nameplates_request ON nameplates (app_id, request_id);

CREATE TABLE nameplate_sides (
	nameplate_id INTEGER REFERENCES nameplates(id) NOT NULL,
	claimed BOOLEAN,
	side VARCHAR,
	added INTEGER
);
CREATE INDEX idx_nameplate_side ON nameplate_sides (nameplate_id, side);
`
