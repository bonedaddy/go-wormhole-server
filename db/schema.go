package db

const schemaVersion = 1

const relaySchema = `
CREATE TABLE version (
	version INTEGER NOT NULL
);
`
