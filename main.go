package main

import (
	"fmt"
	"github.com/cloudbees-compliance/compliance-hub-plugin-jenkins-master/jenkinsmaster"
	"github.com/deliveryblueprints/chplugin-service-go/plugin"
	"github.com/spf13/viper"
	"net"
	"time"

	"github.com/cloudbees-compliance/compliance-hub-plugin-jenkins-master/config"
	"github.com/deliveryblueprints/chlog-go/log"
	service "github.com/deliveryblueprints/chplugin-go/v0.4.0/servicev0_4_0"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
)

func main() {
	config.InitConfig()
	trackingInfo := map[string]string{"Service": "Jenkins-Master-Plugin"}
	log.Init(viper.GetViper(), trackingInfo)
	netListener := getNetListener(viper.GetString("server.address"), viper.GetUint("server.port"))

	gRPCServer := grpc.NewServer(grpc.MaxRecvMsgSize(viper.GetInt("grpc.maxrecvsize")),
		grpc.UnaryInterceptor(otelgrpc.UnaryServerInterceptor()),
		grpc.StreamInterceptor(otelgrpc.StreamServerInterceptor()))

	chPluginService := plugin.CHPluginServiceBuilder(
		jenkinsmaster.NewJenkinsMasterService(),
		viper.GetInt("service.workerpool.size"),
		int64(viper.GetInt("heartbeat.timer")),
	)
	service.RegisterCHPluginServiceServer(gRPCServer, chPluginService)
	log.Info().Msgf("Starting: %s", time.Now().Format(time.RFC3339))
	// start the server
	if err := gRPCServer.Serve(netListener); err != nil {
		log.Panic().Err(err).Msg("failed to serve")
	}
}

func getNetListener(host string, port uint) net.Listener {
	log.Info().Msgf("Binding gRPC server on %s:%d", host, port)
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		log.Panic().Err(err).Msg("failed to listen")
	}

	return lis
}
