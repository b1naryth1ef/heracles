package heracles

import (
	"log"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/b1naryth1ef/heracles/db"
	"github.com/spf13/viper"
)

func Run() {
	rand.Seed(time.Now().UTC().UnixNano())

	db.InitDB(viper.GetString("db_path"), viper.GetString("secret_key"))

	router := NewRouter()

	bind := viper.GetString("bind")
	if strings.HasPrefix(bind, "unix://") {
		listener, err := net.Listen("unix", strings.TrimPrefix(bind, "unix://"))
		if err != nil {
			panic(err)
		}
		defer listener.Close()

		server := http.Server{
			Handler: router,
		}

		log.Printf("Listening on %v", bind)
		server.Serve(listener)
	} else {
		log.Printf("Listening on %v", bind)
		log.Fatalln(http.ListenAndServe(bind, router))
	}
}
