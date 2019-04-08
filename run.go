package heracles

import (
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/b1naryth1ef/heracles/db"
	"github.com/spf13/viper"
)

func Run() {
	rand.Seed(time.Now().UTC().UnixNano())

	db.InitDB(viper.GetString("db_path"), viper.GetString("secret_key"))

	router := NewRouter()
	log.Printf("Listening on %v", viper.GetString("bind"))
	log.Fatalln(http.ListenAndServe(viper.GetString("bind"), router))
}
