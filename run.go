package heracles

import (
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/spf13/viper"
)

func Run() {
	rand.Seed(time.Now().UTC().UnixNano())

	initDB(viper.GetString("db_path"), viper.GetString("secret_key"))

	router := NewRouter()
	log.Printf("Listening on %v", viper.GetString("bind"))
	log.Fatalln(http.ListenAndServe(viper.GetString("bind"), router))
}
