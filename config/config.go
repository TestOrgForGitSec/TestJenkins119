package config

import (
	"github.com/cloudbees-compliance/go-common/secretsmanager"
	chstring "github.com/cloudbees-compliance/go-common/strings"
	"github.com/rs/zerolog/log"
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

	// demo stuff
	viper.SetDefault("demo.account.filter", "")
	viper.SetDefault("demo.asset.filter", "")

	_ = viper.BindEnv("aws.region", "AWS_REGION")         // err will be ignored
	_ = viper.BindEnv("secret.manager", "SECRET_MANAGER") // err will be ignored
	readSecrets(viper.GetViper())
}

func readSecrets(config *viper.Viper) {
	source := config.GetString("secret.manager")

	if !chstring.IsEmpty(&source) {
		reader := secretsmanager.GetReader(source)
		if reader != nil {
			secureConfigs, err := reader.Read()
			if err != nil {
				log.Error().Err(err).Msgf("Failed to use secret manager %v", err)
			} else {
				err = config.MergeConfigMap(secureConfigs)
				if err != nil {
					log.Error().Err(err).Msgf("Failed to update secret config %v", err)
				}
			}
		}
	}
}
