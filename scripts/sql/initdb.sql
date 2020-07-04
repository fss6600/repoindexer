-- Скрипт инициализации БД
DROP TABLE IF EXISTS files;
DROP TABLE IF EXISTS info;
DROP TABLE IF EXISTS packages;
DROP TABLE IF EXISTS aliases;
DROP TABLE IF EXISTS excludes;

-- Пакеты подсистем
CREATE TABLE packages
(
    id    INTEGER PRIMARY KEY AUTOINCREMENT,
    name  VARCHAR     NOT NULL UNIQUE,
    hash  VARCHAR(40) NOT NULL,
    size  INTEGER DEFAULT 0,
    fcnt  INTEGER DEFAULT 0
);
CREATE UNIQUE INDEX idx_packages
    ON packages (name);

-- файлы в пакетах
CREATE TABLE files
(
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    package_id INTEGER     NOT NULL,
    path       VARCHAR     NOT NULL,
    size       INTEGER     NOT NULL,
    mdate      INTEGER     NOT NULL,
    hash       VARCHAR(40) NOT NULL,
    FOREIGN KEY (package_id) REFERENCES packages (id)
        ON DELETE CASCADE
        ON UPDATE CASCADE
);
CREATE INDEX idx_file_path
    ON files (path);

-- информация о БД
CREATE TABLE info
(
    id         INTEGER PRIMARY KEY,
    vers_major INTEGER NOT NULL,
    vers_minor INTEGER NOT NULL
);

-- псевдонимы пакетов подсистем
CREATE TABLE aliases
(
    alias VARCHAR NOT NULL UNIQUE ,
    name  VARCHAR NOT NULL UNIQUE
);
CREATE UNIQUE INDEX idx_alias
    ON aliases (name, alias);

-- заблокированные пакеты подсистем
CREATE TABLE excludes
(
    name VARCHAR NOT NULL UNIQUE
);
CREATE INDEX idx_excludes
    ON excludes (name);
