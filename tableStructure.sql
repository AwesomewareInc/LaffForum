-- Describe POSTS
CREATE TABLE "posts" (
    "id" INTEGER PRIMARY KEY,
    "topic" INTEGER,
    "subject" TEXT,
    "contents" TEXT,
    "author" INTEGER,
    "replyto" INTEGER,
    "timestamp" INTEGER
    PRIMARY KEY("id" AUTOINCREMENT)
)

-- Describe SECTIONS
CREATE TABLE "sections" (
    "id"    INTEGER,
    "name"  TEXT,
    "adminonly" INTEGER DEFAULT 0,
    PRIMARY KEY("id" AUTOINCREMENT)
)

-- Describe USERS
CREATE TABLE "users" (
    "id"    INTEGER,
    "username"  TEXT,
    "password"  TEXT,
    "prettyname"    TEXT,
    "timestamp" TEXT,
    "bio"   TEXT, admin INTEGER,
    PRIMARY KEY("id" AUTOINCREMENT)
)
