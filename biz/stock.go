package biz

import (
	"context"
	"errors"
	"fmt"
	"github.com/i-coder-robot/mic-trainning-lessons-part3/custom_error"
	"github.com/i-coder-robot/mic-trainning-lessons-part3/internal"
	"github.com/i-coder-robot/mic-trainning-lessons-part3/model"
	"github.com/i-coder-robot/mic-trainning-lessons-part3/proto/pb"
	"google.golang.org/protobuf/types/known/emptypb"
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
