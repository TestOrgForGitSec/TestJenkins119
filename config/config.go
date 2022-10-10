package config

import (
	"strings"

	"github.com/spf13/viper"
)

func InitConfig() {
	viper.SetEnvPrefix("ch")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	viper.SetDefault("server.address", "127.0.0.1")
	viper.SetDefault("server.port", 5017)

	// dev stuff
	viper.SetDefault("db.log.level", "debug")
	viper.SetDefault("log.colour", false)
	viper.SetDefault("log.callerinfo", false)
	viper.SetDefault("log.useconsolewriter", false)
	viper.SetDefault("log.unixtime", false)
	viper.SetDefault("log.level", "debug")
}
