package db

import (
	"database/sql"
	"log"

	"github.com/bwmarrin/go-alone"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/viper"
)

var difficulty int
var db *sqlx.DB
var signer *goalone.Sword

type Bits uint64

func (b Bits) Set(flag Bits) Bits   { return b | flag }
func (b Bits) Clear(flag Bits) Bits { return b &^ flag }
func (b Bits) Has(flag Bits) bool   { return b&flag != 0 }

const (
	USER_FLAG_ADMIN = 1 << iota
)

func InitDB(path, secretKey string) {
	difficulty = viper.GetInt("difficulty")

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
