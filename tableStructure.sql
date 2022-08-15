-- Describe POSTS
CREATE TABLE IF NOT EXISTS "posts" (
    "id" INTEGER PRIMARY KEY,
    "topic" INTEGER,
    "subject" TEXT,
    "contents" TEXT,
    "author" INTEGER,
    "replyto" INTEGER,
    "timestamp" INTEGER,
    "deleted" INTEGER,
    PRIMARY KEY("id" AUTOINCREMENT)
)

-- Describe SECTIONS
CREATE TABLE IF NOT EXISTS "sections" (
    "id"    INTEGER,
    "name"  TEXT,
    "adminonly" INTEGER DEFAULT 0,
    PRIMARY KEY("id" AUTOINCREMENT)
)

-- Describe USERS
CREATE TABLE IF NOT EXISTS "users" (
    "id"    INTEGER,
    "username"  TEXT,
    "password"  TEXT,
    "prettyname"    TEXT,
    "timestamp" TEXT,
    "bio"   TEXT, admin INTEGER,
    PRIMARY KEY("id" AUTOINCREMENT)
)
