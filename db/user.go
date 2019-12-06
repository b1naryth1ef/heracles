package db

import (
	"strconv"

	"golang.org/x/crypto/bcrypt"
)

const (
	USER_FLAG_ADMIN = 1 << iota
)

const USER_SCHEMA = `
CREATE TABLE IF NOT EXISTS users (
	id INTEGER PRIMARY KEY,
	username TEXT,
	password TEXT,
	flags INTEGER,
	discord_id INTEGER
);
`

type User struct {
	Id        int64  `json:"id" db:"id"`
	Username  string `json:"username" db:"username"`
	Password  string `json:"-" db:"password"`
	Flags     Bits   `json:"-" db:"flags"`
	DiscordId *int64 `json:"discord_id" db:"discord_id"`
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

func (u *User) UpdatePassword(password string) error {
	var passwordHash string

	if password != "" {
		passwordHashRaw, err := bcrypt.GenerateFromPassword([]byte(password), difficulty)
		if err != nil {
			return err
		}
		passwordHash = string(passwordHashRaw)
	}

	_, err := db.Exec(
		`UPDATE users SET password=? WHERE id=?`,
		passwordHash,
		u.Id,
	)
	return err
}

func CreateUser(username, password string, flags Bits, discordId *int64) (*User, error) {
	var passwordHash string
	if password != "" {
		passwordHashRaw, err := bcrypt.GenerateFromPassword([]byte(password), difficulty)
		if err != nil {
			return nil, err
		}
		passwordHash = string(passwordHashRaw)
	}

	result, err := db.Exec(
		`INSERT INTO users (username, password, flags, discord_id) VALUES (?, ?, ?, ?);`,
		username,
		passwordHash,
		flags,
		discordId,
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &User{
		Id:        id,
		Username:  username,
		Password:  password,
		Flags:     flags,
		DiscordId: discordId,
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

func GetUserByDiscordId(id int64) (*User, error) {
	var user User

	err := db.Get(&user, `SELECT * FROM users WHERE discord_id=?`, id)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func GetUserByToken(token string, isAPI bool) (*User, error) {
	var user User

	var err error
	if isAPI {
		err = db.Get(&user, `
			SELECT u.* FROM users u
			JOIN user_tokens ut ON u.id = ut.user_id
			WHERE ut.token = ? AND ut.flags & ?
		`, token, USER_TOKEN_FLAG_API)
	} else {
		err = db.Get(&user, `
			SELECT u.* FROM users u
			JOIN user_tokens ut ON u.id = ut.user_id
			WHERE ut.token = ?
		`, token)
	}

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
	if users == nil {
		return make([]User, 0), err
	}
	return users, err
}
