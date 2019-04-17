package db

import (
	"strconv"

	"golang.org/x/crypto/bcrypt"
)

const USER_SCHEMA = `
CREATE TABLE IF NOT EXISTS users (
	id INTEGER PRIMARY KEY,
	username TEXT,
	password TEXT,
	flags INTEGER
);
`

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

func CreateUser(username, password string, flags Bits) (*User, error) {
	var passwordHash string
	if password != "" {
		passwordHashRaw, err := bcrypt.GenerateFromPassword([]byte(password), difficulty)
		if err != nil {
			return nil, err
		}
		passwordHash = string(passwordHashRaw)
	}

	result, err := db.Exec(
		`INSERT INTO users (username, password, flags) VALUES (?, ?, ?);`,
		username,
		passwordHash,
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
	if users == nil {
		return make([]User, 0), err
	}
	return users, err
}
