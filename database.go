package heracles

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
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

func (b Bits) Set(flag Bits) Bits   { return b | flag }
func (b Bits) Clear(flag Bits) Bits { return b &^ flag }
func (b Bits) Has(flag Bits) bool   { return b&flag != 0 }

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

func (u *User) IsAdmin() bool {
	return u.Flags.Has(USER_FLAG_ADMIN)
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
		JOIN user_tokens ut ON u.id = ut.user_id
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

func GetUsers() ([]User, error) {
	var users []User
	err := db.Select(&users, `SELECT * FROM users`)
	return users, err
}

type UserToken struct {
	Id     int64  `json:"id" db:"id"`
	UserId int64  `json:"-" db:"user_id"`
	Name   string `json:"name" db:"name"`
	Token  string `json:"token" db:"token"`
}

func (ut *UserToken) Delete() error {
	_, err := db.Exec(`DELETE FROM user_tokens WHERE id=?`, ut.Id)
	return err
}

func (ut *UserToken) Save() error {
	_, err := db.Exec(
		`UPDATE user_tokens SET name=? AND token=? WHERE id=?`,
		ut.Name,
		ut.Token,
		ut.Id,
	)
	return err
}

func GenerateUserTokenContents() (string, error) {
	tokenRaw := make([]byte, 128)
	_, err := rand.Read(tokenRaw)
	if err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(tokenRaw), nil
}

func CreateUserToken(userId int64, name string) (*UserToken, error) {
	tokenEncoded, err := GenerateUserTokenContents()
	if err != nil {
		return nil, err
	}

	result, err := db.Exec(
		`INSERT INTO user_tokens (user_id, name, token) VALUES (?, ?, ?);`,
		userId,
		name,
		tokenEncoded,
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &UserToken{
		Id:     id,
		UserId: userId,
		Name:   name,
		Token:  tokenEncoded,
	}, nil
}

func GetUserTokenById(id int64) (*UserToken, error) {
	var userToken UserToken
	err := db.Get(&userToken, `SELECT * FROM user_tokens WHERE id=?`, id)
	if err != nil {
		return nil, err
	}
	return &userToken, nil
}

func GetUserTokensByUserId(id int64) ([]UserToken, error) {
	var userTokens []UserToken
	err := db.Select(&userTokens, `SELECT * FROM user_tokens WHERE user_id=?`, id)
	return userTokens, err
}

type Realm struct {
	Id   int64  `json:"id" db:"id"`
	Name string `json:"name" db:"name"`
}

func CreateRealm(name string) (*Realm, error) {
	result, err := db.Exec(`INSERT INTO realms (name) VALUES (?);`, name)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &Realm{
		Id:   id,
		Name: name,
	}, nil
}

func GetRealmById(id int64) (*Realm, error) {
	var realm Realm
	err := db.Get(&realm, `SELECT * FROM realms WHERE id=?`, id)
	if err != nil {
		return nil, err
	}

	return &realm, nil
}

func GetRealms() ([]Realm, error) {
	var realms []Realm
	err := db.Select(&realms, `SELECT * FROM realms`)
	return realms, err
}

type UserRealmGrant struct {
	UserId  int64   `json:"user_id" db:"user_id"`
	RealmId int64   `json:"realm_id" db:"realm_id"`
	Alias   *string `json:"alias" db:"alias"`
}

func CreateUserRealmGrant(userId int64, realmId int64, alias *string) (*UserRealmGrant, error) {
	_, err := db.Exec(`
		INSERT INTO user_realm_grants (user_id, realm_id, alias)
		VALUES (?, ?, ?);
	`, userId, realmId, alias)
	if err != nil {
		return nil, err
	}

	return &UserRealmGrant{
		UserId:  userId,
		RealmId: realmId,
		Alias:   alias,
	}, nil
}

func GetUserRealmGrantByRealmName(userId int64, realmName string) (*UserRealmGrant, error) {
	var grant UserRealmGrant

	err := db.Get(&grant, `
		SELECT urg.* FROM user_realm_grants urg
		JOIN realms r ON r.id = urg.realm_id
		WHERE r.name = ? AND urg.user_id = ?
	`, realmName, userId)
	if err != nil {
		return nil, err
	}

	return &grant, nil
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
	flags = flags.Set(USER_FLAG_ADMIN)

	_, err := CreateUser("admin", "admin", flags)
	if err != nil {
		panic(err)
	}
}
