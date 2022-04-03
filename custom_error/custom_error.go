package custom_error

const (
	StockNotFound   = "库存不存在"
	StockNotEnough  = "库存不足"
	ParamError      = "参数错误"
	RedisLockErr    = "Redis分布式锁错误"
	CreateDetailErr = "创建产品详情失败"
)
