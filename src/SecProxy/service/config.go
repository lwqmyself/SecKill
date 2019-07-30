package service

import "sync"

const (
	ProductStatusNormal       = 0
	ProductStatusSaleOut      = 1
	ProductStatusForceSaleOut = 2
)

type RedisConf struct {
	RedisAddr        string
	RedisMaxIdle     int
	RedisMaxActive   int
	RedisIdleTimeout int
}

type EtcdConf struct {
	EtcdAddr          string
	Timeout           int
	EtcdSecKeyPrefix  string
	EtcdSecProductKey string
}

type SecKillConf struct {
	RedisConf         RedisConf
	EtcdConf          EtcdConf
	LogPath           string
	LogLevel          string
	SecProductInfoMap map[int]*SecProductInfoConf
	RwSecProductLock  sync.RWMutex
}

type SecProductInfoConf struct {
	//大写是为了json反序列化
	ProductId int
	StartTime int64
	EndTime   int64
	Status    int
	Total     int
	Left      int
}
