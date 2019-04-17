package main

import (
	"fmt"
	"strings"

	"github.com/b1naryth1ef/heracles"
	"github.com/spf13/viper"
)

func main() {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)

	err := viper.ReadInConfig()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
	}

	heracles.Run()
}
