CREATE TABLE IF NOT EXISTS notes (
    id blob NOT NULL PRIMARY KEY,
    title text NOT NULL,
	content text NOT NULL
);