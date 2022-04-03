package main

import (
	"fmt"
	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/google/uuid"
	"github.com/hashicorp/consul/api"
	"github.com/i-coder-robot/mic-trainning-lessons-part3/biz"
	"github.com/i-coder-robot/mic-trainning-lessons-part3/internal"
	"github.com/i-coder-robot/mic-trainning-lessons-part3/proto/pb"
	"github.com/i-coder-robot/mic-trainning-lessons-part3/util"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"net"
)

func main() {
	port, err := util.GenRandomPort()
	if err != nil {
		zap.S().Error(err)
		panic(err)
	}
	if internal.AppConf.Debug {
		port = internal.AppConf.StockSrvConfig.Port
	}
	addr := fmt.Sprintf("%s:%d", internal.AppConf.StockSrvConfig.Host, port)
	server := grpc.NewServer()
	pb.RegisterStockServiceServer(server, &biz.StockServer{})
	listen, err := net.Listen("tcp", addr)
	if err != nil {
		zap.S().Error("stock_srv启动异常:" + err.Error())
		panic(err)
	}
	grpc_health_v1.RegisterHealthServer(server, health.NewServer())
	defaultConfig := api.DefaultConfig()
	defaultConfig.Address = fmt.Sprintf("%s:%d",
		internal.AppConf.ConsulConfig.Host,
		internal.AppConf.ConsulConfig.Port)
	client, err := api.NewClient(defaultConfig)
	if err != nil {
		panic(err)
	}
	checkAddr := fmt.Sprintf("%s:%d", internal.AppConf.StockSrvConfig.Host, port)
	check := &api.AgentServiceCheck{
		GRPC:                           checkAddr,
		Timeout:                        "3s",
		Interval:                       "1s",
		DeregisterCriticalServiceAfter: "5s",
	}
	randUUID := uuid.New().String()
	reg := api.AgentServiceRegistration{
		Name:    internal.AppConf.StockSrvConfig.SrvName,
		ID:      randUUID,
		Port:    port,
		Tags:    internal.AppConf.StockSrvConfig.Tags,
		Address: internal.AppConf.StockSrvConfig.Host,
		Check:   check,
	}
	err = client.Agent().ServiceRegister(&reg)
	if err != nil {
		panic(err)
	}
	fmt.Println(fmt.Sprintf("%s启动在%d", randUUID, port))
	pushConsumer, _ := rocketmq.NewPushConsumer(
		consumer.WithNameServer([]string{"192.168.0.107:9876"}),
		consumer.WithGroupName("HappyStockGroup"),
	)
	pushConsumer.Subscribe("Happy_BackStockTopic", consumer.MessageSelector{}, biz.BackStock)

	err = server.Serve(listen)
	if err != nil {
		panic(err)
	}
}
