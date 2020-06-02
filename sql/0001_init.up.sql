PRAGMA foreign_keys = ON;

CREATE TABLE videos
(
    id          INTEGER PRIMARY KEY ASC AUTOINCREMENT,
    video_id    TEXT UNIQUE NOT NULL,
    title       TEXT,
    uploader    TEXT,
    upload_date DATE,
    info        JSON,
    status      TEXT CHECK (status IN ('NEW', 'SKIPPED', 'FAILED', 'DOWNLOADED'))
                        DEFAULT 'NEW',
    volume      TEXT,
    path        TEXT,
    filesize    INTEGER,
    filehash    TEXT,
    attempts    INTEGER DEFAULT 0
);

CREATE INDEX ytbackup__status_idx ON videos (status);
