-- Describe POSTS
CREATE TABLE IF NOT EXISTS "posts" (
    "id" INTEGER PRIMARY KEY,
    PRIMARY KEY("id" AUTOINCREMENT)
);

ALTER TABLE "posts" ADD "topic"             INTEGER;
ALTER TABLE "posts" ADD "subject"           TEXT;
ALTER TABLE "posts" ADD "contents"          TEXT;
ALTER TABLE "posts" ADD "author"            INTEGER;
ALTER TABLE "posts" ADD "replyto"           INTEGER;
ALTER TABLE "posts" ADD "timestamp"         INTEGER;
ALTER TABLE "posts" ADD "deleted"           INTEGER DEFAULT (0);

-- Describe SECTIONS
CREATE TABLE IF NOT EXISTS "sections" (
    "id" INTEGER PRIMARY KEY,
    PRIMARY KEY("id" AUTOINCREMENT)
);

ALTER TABLE "posts" ADD "name"              TEXT;
ALTER TABLE "posts" ADD "adminonly"         INTEGER DEFAULT 0;

-- Describe USERS
CREATE TABLE IF NOT EXISTS "users" (
    "id" INTEGER PRIMARY KEY,
    PRIMARY KEY("id" AUTOINCREMENT)
);

ALTER TABLE "users" ADD "id"                INTEGER;
ALTER TABLE "users" ADD "username"          TEXT;
ALTER TABLE "users" ADD "password"          TEXT;
ALTER TABLE "users" ADD "prettyname"        TEXT;
ALTER TABLE "users" ADD "timestamp"         TEXT;
ALTER TABLE "users" ADD "bio"               TEXT;
ALTER TABLE "users" ADD "admin"             INTEGER;

CREATE TABLE IF NOT EXISTS "sessions" (
    "id" INTEGER PRIMARY KEY,
    PRIMARY KEY("id" AUTOINCREMENT)
);

ALTER TABLE "sessions" ADD "genkey"         TEXT;
ALTER TABLE "sessions" ADD "pubkey"         TEXT;
ALTER TABLE "sessions" ADD "privkey"        TEXT;
ALTER TABLE "sessions" ADD "username"       TEXT;
ALTER TABLE "sessions" ADD "timestamp"      TEXT;
