package db

import (
	"crypto/rand"
	"encoding/base64"
)

const USER_TOKEN_SCHEMA = `
CREATE TABLE IF NOT EXISTS user_tokens (
	id INTEGER PRIMARY KEY,
	user_id INTEGER,
	name TEXT,
	token TEXT
);
`

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

	userToken := &UserToken{
		UserId: userId,
		Name:   name,
		Token:  tokenEncoded,
	}

	userToken.Id, err = result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return userToken, nil
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
	if userTokens == nil {
		return make([]UserToken, 0), err
	}
	return userTokens, err
}
