package heracles

import (
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/b1naryth1ef/heracles/db"
	"github.com/spf13/viper"
)

func Run() {
	rand.Seed(time.Now().UTC().UnixNano())

	db.InitDB(viper.GetString("db.path"), viper.GetString("security.secret"), viper.GetInt("security.bcrypt.difficulty"))

	router := NewRouter()

	bind := viper.GetString("web.bind")
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
		os.Stderr.WriteString("READY")
		server.Serve(listener)
	} else {
		log.Printf("Listening on %v", bind)
		os.Stderr.WriteString("READY")
		log.Fatalln(http.ListenAndServe(bind, router))
	}
}
