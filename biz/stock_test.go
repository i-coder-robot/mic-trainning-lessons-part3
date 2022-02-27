package biz

import (
	"context"
	"fmt"
	"github.com/i-coder-robot/mic-trainning-lessons-part3/internal"
	"github.com/i-coder-robot/mic-trainning-lessons-part3/model"
	"github.com/i-coder-robot/mic-trainning-lessons-part3/proto/pb"
	"google.golang.org/grpc"
	"sync"
	"testing"
)

var client pb.StockServiceClient

func init() {
	addr := fmt.Sprintf("%s:%d", internal.AppConf.StockSrvConfig.Host,
		internal.AppConf.StockSrvConfig.Port)
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	client = pb.NewStockServiceClient(conn)
}

func TestStockServer_SetStock(t *testing.T) {
	_, err := client.SetStock(context.Background(), &pb.ProductStockItem{
		ProductId: 1,
		Num:       200,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestStockServer_StockDetail(t *testing.T) {
	detail, err := client.StockDetail(context.Background(), &pb.ProductStockItem{
		ProductId: 99,
	})
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(detail)
}

var wg sync.WaitGroup

func TestStockServer_Sell(t *testing.T) {
	//item:=&pb.ProductStockItem{
	//	ProductId: 1,
	//	Num:       10,
	//}
	//var itemList []*pb.ProductStockItem
	//itemList=append(itemList, item)
	// sellItem:= &pb.SellItem{
	//	 StockItemList:itemList,
	//}
	//res, err := client.Sell(context.Background(), sellItem)
	//if err != nil {
	//	t.Fatal(err)
	//}
	//fmt.Println(res)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			var itemList []*pb.ProductStockItem
			item := &pb.ProductStockItem{
				ProductId: 1,
				Num:       1,
			}
			itemList = append(itemList, item)
			sellItem := &pb.SellItem{
				StockItemList: itemList,
			}
			res, err := client.Sell(context.Background(), sellItem)
			if err != nil {
				t.Fatal(err)
			}
			fmt.Println(res)
			wg.Done()
		}()
	}
	wg.Wait()
}

func TestStockServer_BackStock(t *testing.T) {
	var itemList []*pb.ProductStockItem
	item := &pb.ProductStockItem{
		ProductId: 1,
		Num:       18,
	}
	itemList = append(itemList, item)
	res, err := client.BackStock(context.Background(), &pb.SellItem{StockItemList: itemList})
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(res)

}

func TestCreateStockItemDetail(t *testing.T) {
	item := model.StockItemDetail{
		OrderNo: "123456789",
		Status:  model.HasSell,
		DetailList: []model.ProductDetail{
			{ProductId: 1, Num: 6},
			{ProductId: 2, Num: 8},
		},
	}
	internal.DB.Save(&item)
}

func TestFindStockItemDetail(t *testing.T) {
	var itemDetail model.StockItemDetail

	internal.DB.Model(model.StockItemDetail{}).Where(model.StockItemDetail{
		OrderNo: "123456789",
	}).First(&itemDetail)
	fmt.Println(itemDetail)
}
