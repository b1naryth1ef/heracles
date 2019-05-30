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
	"layeh.com/radius"
)

func Run() {
	rand.Seed(time.Now().UTC().UnixNano())

	db.InitDB(viper.GetString("db.path"), viper.GetString("security.secret"), viper.GetInt("security.bcrypt.difficulty"))

	if viper.GetBool("radius.enabled") {
		server := radius.PacketServer{
			Handler:      radius.HandlerFunc(handleRadiusRequest),
			SecretSource: radius.StaticSecretSource([]byte(viper.GetString("radius.secret"))),
		}

		go func() {
			log.Printf("RADIUS listening on :1812")
			if err := server.ListenAndServe(); err != nil {
				log.Fatal(err)
			}
		}()
	}

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
		server.Serve(listener)
	} else {
		log.Printf("Listening on %v", bind)
		log.Fatalln(http.ListenAndServe(bind, router))
	}
}
