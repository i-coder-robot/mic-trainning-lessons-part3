package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/google/uuid"
	"github.com/hashicorp/consul/api"
	"github.com/i-coder-robot/mic-trainning-lessons-part3/biz"
	"github.com/i-coder-robot/mic-trainning-lessons-part3/internal"
	"github.com/i-coder-robot/mic-trainning-lessons-part3/model"
	"github.com/i-coder-robot/mic-trainning-lessons-part3/proto/pb"
	"github.com/i-coder-robot/mic-trainning-lessons-part3/util"
	"gorm.io/gorm"

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
	if !internal.AppConf.Debug {
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
		consumer.WithNameServer([]string{"192.168.0.104:9876"}),
		consumer.WithGroupName("HappyStockGroup"),
	)
	pushConsumer.Subscribe("Happy_BackStockTopic", consumer.MessageSelector{},
		func(ctx context.Context,
			messageExt ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
			for i := range messageExt {
				var order model.Order
				err := json.Unmarshal(messageExt[i].Body, &order)
				if err != nil {
					zap.S().Error("Unmarshal Error:" + err.Error())
					return consumer.ConsumeSuccess, nil
				}
				tx := internal.DB.Begin()
				var detail model.StockItemDetail
				r := tx.Where(&model.StockItemDetail{
					OrderNo: order.OrderNo,
					Status:  model.HasSell,
				}).First(&detail)
				if r.RowsAffected < 1 {
					return consumer.ConsumeSuccess, nil
				}
				for _, item := range detail.DetailList {
					ret := tx.Model(&model.Stock{ProductId: item.ProductId}).Update(
						"num", gorm.Expr("num+?", item.Num),
					)
					if ret.RowsAffected < 1 {
						return consumer.ConsumeRetryLater, nil
					}
				}
				result := tx.Model(&model.StockItemDetail{}).
					Where(&model.StockItemDetail{OrderNo: order.OrderNo}).
					Update("status", model.HasBack)
				if result.RowsAffected < 1 {
					tx.Rollback()
					return consumer.ConsumeRetryLater, nil
				}
				tx.Commit()
				return consumer.ConsumeSuccess, nil
			}
			return consumer.ConsumeSuccess, nil
		})

	err = server.Serve(listen)
	if err != nil {
		panic(err)
	}
}
