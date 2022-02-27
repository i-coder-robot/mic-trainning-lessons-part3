package internal

import (
	"encoding/json"
	"fmt"
	goredislib "github.com/go-redis/redis/v8"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v8"
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"github.com/spf13/viper"
	"os"
)

var (
	AppConf   AppConfig
	NacosConf NacosConfig
	Redsync   *redsync.Redsync
)

//var fileName = "dev-config.yaml"

func initNacos() {
	v := viper.New()
	//设置配置文件的名字
	v.SetConfigName("config")
	v.AddConfigPath("$GOPATH/src/mic-trainning-lesson/stock/")
	v.SetConfigType("yaml")
	err := v.ReadInConfig()
	if err != nil {
		panic(err)
	}
	v.Unmarshal(&NacosConf)
}

func initFromNacos() {
	serverConfigs := []constant.ServerConfig{
		{
			IpAddr: NacosConf.Host,
			Port:   NacosConf.Port,
		},
	}
	clientConfig := constant.ClientConfig{
		NamespaceId:         NacosConf.NameSpace,
		TimeoutMs:           5000,
		NotLoadCacheAtStart: true,
		LogDir:              "nacos/log",
		CacheDir:            "nacos/cache",
		RotateTime:          "1h",
		MaxAge:              3,
		LogLevel:            "debug",
	}
	configClient, err := clients.CreateConfigClient(map[string]interface{}{
		"serverConfigs": serverConfigs,
		"clientConfig":  clientConfig,
	})
	if err != nil {
		panic(err)
	}

	content, err := configClient.GetConfig(vo.ConfigParam{
		DataId: NacosConf.DataId,
		Group:  NacosConf.Group,
	})

	if err != nil {
		panic(err)
	}
	json.Unmarshal([]byte(content), &AppConf)
}

func init() {
	getwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	fmt.Println(`%GOPATH/src/`)
	fmt.Println(getwd)
	initNacos()
	initFromNacos()
	fmt.Println("初始化完成...")
	InitRedis()
	InitDB()

	redisAddr := fmt.Sprintf("%s:%d", AppConf.RedisConfig.Host, AppConf.RedisConfig.Port)
	client := goredislib.NewClient(&goredislib.Options{
		Addr: redisAddr,
	})
	pool := goredis.NewPool(client)
	Redsync = redsync.New(pool)
}
