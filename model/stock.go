package model

import (
	"database/sql/driver"
	"encoding/json"
	"gorm.io/gorm"
	"time"
)

type OrderStatus int32

const (
	HasSell OrderStatus = 1
	HasBack OrderStatus = 2
)

type ProductDetail struct {
	ProductId int32
	Num       int32
}

type ProductDetailList []ProductDetail

func (p ProductDetailList) Value() (driver.Value, error) {
	return json.Marshal(p)
}

func (p ProductDetailList) Scan(value interface{}) error {
	return json.Unmarshal(value.([]byte), &p)
}

type StockItemDetail struct {
	OrderNo    string            `gorm:"type:varchar(128),index:order_no,unique"`
	Status     OrderStatus       `gorm:"type:int"`
	DetailList ProductDetailList `gorm:"varchar(128)"`
}

type BaseModel struct {
	ID        int32          `gorm:"primary_key"`
	CreatedAt *time.Time     `gorm:"column:add_time"`
	UpdatedAt *time.Time     `gorm:"column:update_time"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type Stock struct {
	BaseModel
	ProductId int32 `gorm:"type:int;index"`
	Num       int32 `gorm:"type:int"`
	Version   int32 `grom:"type:int"`
}
