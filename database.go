package heracles

import (
	"database/sql"
	"log"
	"strconv"

	"github.com/bwmarrin/go-alone"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

var db *sqlx.DB
var signer *goalone.Sword

const USER_SCHEMA = `
CREATE TABLE IF NOT EXISTS users (
	id INTEGER PRIMARY KEY,
	username TEXT,
	password TEXT,
	flags INTEGER
);
`

const USER_TOKEN_SCHEMA = `
CREATE TABLE IF NOT EXISTS user_tokens (
	id INTEGER PRIMARY KEY,
	user_id INTEGER,
	name TEXT,
	token TEXT
);
`

const REALM_SCHEMA = `
CREATE TABLE IF NOT EXISTS realms (
	id INTEGER PRIMARY KEY,
	name TEXT
);
`

const USER_REALM_GRANT_SCHEMA = `
CREATE TABLE IF NOT EXISTS user_realm_grants (
	user_id INTEGER,
	realm_id INTEGER,
	alias TEXT,

	PRIMARY KEY (user_id, realm_id)
);
`

type Bits uint64

func (b Bits) Set(flag Bits) Bits    { return b | flag }
func (b Bits) Clear(flag Bits) Bits  { return b &^ flag }
func (b Bits) Toggle(flag Bits) Bits { return b ^ flag }
func (b Bits) Has(flag Bits) bool    { return b&flag != 0 }

const (
	USER_FLAG_ADMIN = 1 << iota
)

type User struct {
	Id       int64  `json:"id" db:"id"`
	Username string `json:"username" db:"username"`
	Password string `json:"-" db:"password"`
	Flags    Bits   `json:"-" db:"flags"`
}

func (u *User) CheckPassword(password string) error {
	return bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
}

func (u *User) GetAuthSecret() []byte {
	return signer.Sign([]byte(strconv.Itoa(int(u.Id))))
}

func CreateUser(username, password string, flags Bits) (*User, error) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), 8)
	if err != nil {
		return nil, err
	}

	result, err := db.Exec(
		`INSERT INTO users (username, password, flags) VALUES (?, ?, ?);`,
		username,
		string(passwordHash),
		flags,
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &User{
		Id:       id,
		Username: username,
		Password: password,
		Flags:    flags,
	}, nil
}

func GetUserByAuthSecret(data []byte) (*User, error) {
	userIdRaw, err := signer.Unsign(data)
	if err != nil {
		return nil, err
	}

	userId, err := strconv.Atoi(string(userIdRaw))
	if err != nil {
		return nil, err
	}

	return GetUserById(int64(userId))
}

func GetUserById(id int64) (*User, error) {
	var user User

	err := db.Get(&user, `SELECT * FROM users WHERE id=?`, id)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func GetUserByToken(token string) (*User, error) {
	var user User

	err := db.Get(&user, `
		SELECT u.* FROM users u
		JOIN user_token ut ON u.id = ut.user_id
		WHERE ut.token = ?
	`, token)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func GetUserByUsername(username string) (*User, error) {
	var user User

	err := db.Get(&user, `SELECT * FROM users WHERE username=?`, username)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

type UserToken struct {
	Id     int64  `json:"id" db:"id"`
	UserId int64  `json:"user_id" db:"user_id"`
	Name   string `json:"name" db:"name"`
	Token  string `json:"token" db:"token"`
}

type Realm struct {
	Id   int64  `json:"id" db:"id"`
	Name string `json:"name" db:"name"`
}

type UserRealmGrant struct {
	UserId  int64  `json:"user_id" db:"user_id"`
	RealmId int64  `json:"realm_id" db:"realm_id"`
	Alias   string `json:"alias" db:"alias"`
}

func initDB(path, secretKey string) {
	signer = goalone.New([]byte(secretKey))

	db = sqlx.MustConnect("sqlite3", path)
	db.MustExec(USER_SCHEMA)
	db.MustExec(USER_TOKEN_SCHEMA)
	db.MustExec(REALM_SCHEMA)
	db.MustExec(USER_REALM_GRANT_SCHEMA)

	var user User
	err := db.Get(&user, `SELECT * FROM users LIMIT 1`)
	if err == sql.ErrNoRows {
		bootstrapDB()
	}
}

func bootstrapDB() {
	log.Printf("Bootstraping Database w/ admin user")

	var flags Bits
	flags.Set(USER_FLAG_ADMIN)
	_, err := CreateUser("admin", "admin", flags)
	if err != nil {
		panic(err)
	}
}
