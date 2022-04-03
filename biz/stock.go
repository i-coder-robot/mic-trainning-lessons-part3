package biz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/i-coder-robot/mic-trainning-lessons-part3/custom_error"
	"github.com/i-coder-robot/mic-trainning-lessons-part3/internal"
	"github.com/i-coder-robot/mic-trainning-lessons-part3/model"
	"github.com/i-coder-robot/mic-trainning-lessons-part3/proto/pb"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/emptypb"
	"gorm.io/gorm"
)

type StockServer struct {
}

func (s StockServer) SetStock(ctx context.Context, req *pb.ProductStockItem) (*emptypb.Empty, error) {
	//参数校验 1 web层，<1 !="" service
	var stock model.Stock
	//req.ProductId ->Product_srv
	internal.DB.Where("product_id=?", req.ProductId).Find(&stock)
	if stock.ID < 1 {
		stock.ProductId = req.ProductId
		//stock.Num=req.Num
	}
	stock.Num = req.Num

	internal.DB.Save(&stock)
	return &emptypb.Empty{}, nil

}

func (s StockServer) StockDetail(ctx context.Context, req *pb.ProductStockItem) (*pb.ProductStockItem, error) {
	var stock model.Stock
	r := internal.DB.Where("product_id=?", req.ProductId).First(&stock)
	if r.RowsAffected < 1 {
		return nil, errors.New(custom_error.ParamError)
	}
	stockPb := ConvertStockModel2Pb(stock)
	return &stockPb, nil
}

func (s StockServer) Sell(ctx context.Context, req *pb.SellItem) (*emptypb.Empty, error) {
	//之前的视频，一定要搞清楚
	//面试必问，mutex锁-》悲观锁-》乐观锁-》分布式锁,重要，重要，重要！
	tx := internal.DB.Begin()
	stockDetail := model.StockItemDetail{
		OrderNo: req.OrderNo, //修改proto文件，增加OrderNo
		Status:  model.HasSell,
	}
	var sellList model.ProductDetailList
	for _, item := range req.StockItemList {
		var stock model.Stock

		mutex := internal.Redsync.NewMutex(fmt.Sprintf("product_%d", item.ProductId))
		err := mutex.Lock()
		if err != nil {
			return nil, errors.New(custom_error.RedisLockErr)
		}
		r := internal.DB.Where("product_id=?", item.ProductId).First(&stock)
		if r.RowsAffected == 0 {
			tx.Rollback()
			return nil, errors.New(custom_error.StockNotFound)
		}
		if stock.Num < item.Num {
			tx.Rollback()
			return nil, errors.New(custom_error.StockNotEnough)
		}
		stock.Num -= item.Num
		tx.Save(&stock)
		ok, err := mutex.Unlock()
		if !ok || err != nil {
			return nil, errors.New(custom_error.StockNotEnough)
		}
		sellList = append(sellList, model.ProductDetail{
			ProductId: item.ProductId,
			Num:       item.Num,
		})
	}
	stockDetail.DetailList = sellList
	r := tx.Create(&stockDetail)
	if r.RowsAffected < 1 {
		tx.Rollback()
		return nil, errors.New(custom_error.CreateDetailErr)
	}
	tx.Commit()
	return &emptypb.Empty{}, nil
}

func (s StockServer) BackStock(ctx context.Context, req *pb.SellItem) (*emptypb.Empty, error) {
	/*
		什么时候出发回滚？
		超时怎么办？
		订单创建失败?
		手动归还
	*/
	tx := internal.DB.Begin()
	for _, item := range req.StockItemList {
		var stock model.Stock
		r := internal.DB.Where("product_id=?", item.ProductId).First(&stock)
		if r.RowsAffected < 1 {
			tx.Rollback()
			return nil, errors.New(custom_error.StockNotFound)
		}
		stock.Num += item.Num
		tx.Save(&stock)
	}
	tx.Commit()
	return &emptypb.Empty{}, nil
}

func ConvertStockModel2Pb(s model.Stock) pb.ProductStockItem {
	stock := pb.ProductStockItem{
		ProductId: s.ProductId,
		Num:       s.Num,
	}
	return stock
}

func BackStock(ctx context.Context, messageExt ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
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
}
