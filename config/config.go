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

	viper.SetDefault("service.workerpool.size", 3)
	viper.SetDefault("heartbeat.timer", 45)

	// 1GB max. recv size on grpc by default
	viper.SetDefault("grpc.maxrecvsize", 1024*1024*1024)

	// dev stuff
	viper.SetDefault("db.log.level", "debug")
	viper.SetDefault("log.colour", false)
	viper.SetDefault("log.callerinfo", false)
	viper.SetDefault("log.useconsolewriter", false)
	viper.SetDefault("log.unixtime", false)
	viper.SetDefault("log.level", "debug")
}
