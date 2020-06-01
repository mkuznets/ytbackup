CREATE TABLE ytbackup
(
    id       INTEGER PRIMARY KEY ASC AUTOINCREMENT,
    video_id TEXT UNIQUE NOT NULL,
    title    TEXT,
    uploader TEXT,
    info     JSON,
    status   TEXT CHECK (status IN ('NEW', 'SKIPPED', 'FAILED', 'DOWNLOADED')) DEFAULT 'NEW',
    volume   TEXT,
    path     TEXT,
    filesize INTEGER
);

CREATE INDEX ytbackup__status_idx ON ytbackup (status);
