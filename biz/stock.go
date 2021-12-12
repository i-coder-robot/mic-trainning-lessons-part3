package biz

import (
	"context"
	"errors"
	"github.com/i-coder-robot/mic-trainning-lessons-part3/custom_error"
	"github.com/i-coder-robot/mic-trainning-lessons-part3/internal"
	"github.com/i-coder-robot/mic-trainning-lessons-part3/model"
	"github.com/i-coder-robot/mic-trainning-lessons-part3/proto/pb"
	"google.golang.org/protobuf/types/known/emptypb"
	"sync"
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

var m sync.Mutex

func (s StockServer) Sell(ctx context.Context, req *pb.SellItem) (*emptypb.Empty, error) {
	tx := internal.DB.Begin()
	m.Lock() //为了防止并发安全，加入互斥锁，但是性能差，要用分布式锁，解决这个问题
	defer m.Unlock()
	for _, item := range req.StockItemList {
		var stock model.Stock
		r := internal.DB.Where("product_id=?", item.ProductId).First(&stock)
		if r.RowsAffected == 0 {
			tx.Rollback()
			return nil, errors.New(custom_error.ProductNotFound)
		}
		if stock.Num < item.Num {
			tx.Rollback()
			return nil, errors.New(custom_error.StockNotEnough)
		}
		stock.Num -= item.Num
		//stock.Num=stock.Num-item.Num
		tx.Save(&stock)
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
			return nil, errors.New(custom_error.ProductNotFound)
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
