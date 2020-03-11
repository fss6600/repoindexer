package obj

const initSQL = `
-- Скрипт инициализации БД
DROP TABLE IF EXISTS files;
DROP TABLE IF EXISTS info;
DROP TABLE IF EXISTS packages;
DROP TABLE IF EXISTS aliases;
DROP TABLE IF EXISTS exludes;

-- Пакеты подсистем
CREATE TABLE packages
(
    id    INTEGER PRIMARY KEY AUTOINCREMENT,
    name  VARCHAR     NOT NULL UNIQUE,
    hash  VARCHAR(40) NOT NULL,
    alias TEXT
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
    mdate      DATETIME    NOT NULL,
    hash       VARCHAR(40) NOT NULL,
    FOREIGN KEY (package_id) REFERENCES packages (id)
        ON DELETE CASCADE
        ON UPDATE CASCADE
);
CREATE UNIQUE INDEX idx_file_path
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
    id    INTEGER PRIMARY KEY AUTOINCREMENT,
    alias VARCHAR NOT NULL,
    name  VARCHAR NOT NULL
);
CREATE UNIQUE INDEX idx_alias
    ON aliases (name, alias);

-- заблокированные пакеты подсистем
CREATE TABLE excludes
(
    id   INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR NOT NULL UNIQUE
);
`
