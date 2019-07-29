package main

import (
	"SecProxy/service"
	"encoding/json"
	"fmt"
	"github.com/astaxie/beego/logs"
	"github.com/gomodule/redigo/redis"
	etcd_client "go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/mvcc/mvccpb"
	"golang.org/x/net/context"
	"time"
)

var (
	redisPool  *redis.Pool
	etcdClient *etcd_client.Client

)

func initRedis() (err error) {
	redisPool = &redis.Pool{
		MaxIdle:     secKillConf.RedisConf.RedisMaxIdle,
		MaxActive:   secKillConf.RedisConf.RedisMaxActive,
		IdleTimeout: time.Duration(secKillConf.RedisConf.RedisIdleTimeout) * time.Second,
		Dial: func() (redis.Conn, error) {
			fmt.Println(secKillConf.RedisConf.RedisAddr)
			return redis.Dial("tcp", secKillConf.RedisConf.RedisAddr)
		},
	}
	fmt.Println(secKillConf.RedisConf.RedisAddr)
	conn := redisPool.Get()
	defer conn.Close()
	_, err = conn.Do("PING")
	if err != nil {
		logs.Error("ping redis failed , err:%v", err)
		return
	}
	return
}

func initEtcd() (err error) {

	cli, err := etcd_client.New(etcd_client.Config{
		Endpoints:   []string{secKillConf.EtcdConf.EtcdAddr},
		DialTimeout: time.Duration(secKillConf.EtcdConf.Timeout) * time.Second,
	})
	if err != nil {
		logs.Error("connect etcd failed, err:", err)
		return
	}

	etcdClient = cli
	return
}

func convertLogLevel(level string) int {
	switch (level) {
	case "debug":
		return logs.LevelDebug
	case "warn":
		return logs.LevelWarn
	case "info":
		return logs.LevelInfo
	case "trace":
		return logs.LevelTrace
	}
	return logs.LevelDebug
}

func initLogs() (err error) {
	config := make(map[string]interface{})
	config["filename"] = secKillConf.LogPath
	config["level"] = convertLogLevel(secKillConf.LogLevel)
	configStr, err := json.Marshal(config)
	if err != nil {
		fmt.Println("marshal failed ,err:%v", err)
		return
	}
	logs.SetLogger(logs.AdapterFile, string(configStr))
	return
}

func loadSecConf() (err error) {
	//key := fmt.Sprintf("%s/product",secKillConf.etcdConf.etcdSecKeyPrefix)
	resp, err := etcdClient.Get(context.Background(), secKillConf.EtcdConf.EtcdSecProductKey)
	if err != nil {
		logs.Error("get [%s] from etcd failed, err : %v", secKillConf.EtcdConf.EtcdSecProductKey, err)
		return
	}
	var secProductInfo []service.SecProductInfoConf
	for k, v := range resp.Kvs {
		logs.Debug("key[%v] value [%v]", k, v)
		err = json.Unmarshal(v.Value, &secProductInfo)
		if err != nil {
			logs.Error("Unmarshal sec productinfo failed err : %v", err)

		}
	}
	updateSecproductInfo(secProductInfo)
	return
}

func updateSecproductInfo(secProductInfo []service.SecProductInfoConf) {
	var tmp map[int]*service.SecProductInfoConf = make(map[int]*service.SecProductInfoConf,1024)

	for _, v := range secProductInfo {
		tmp[v.ProductId] = &v
	}
	secKillConf.RwSecProductLock.Lock()
	secKillConf.SecProductInfoMap = tmp
	secKillConf.RwSecProductLock.Unlock()
}

func initSecProductWatcher() {
	go watchSecProductKey(secKillConf.EtcdConf.EtcdSecProductKey)
}
func watchSecProductKey(key string) {

	cli, err := etcd_client.New(etcd_client.Config{
		Endpoints:   []string{"localhost:2379", "localhost:22379", "localhost:32379"},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		logs.Error("connect etcd failed, err:", err)
		return
	}

	logs.Debug("begin watch key:%s", key)
	for {
		rch := cli.Watch(context.Background(), key)
		var secProductInfo []service.SecProductInfoConf
		var getConfSucc = true
		for wresp := range rch {
			for _, ev := range wresp.Events {
				if ev.Type == mvccpb.DELETE {
					logs.Warn("key[%s] 's config deleted", key)
					continue
				}

				if ev.Type == mvccpb.PUT && string(ev.Kv.Key) == key {
					err = json.Unmarshal(ev.Kv.Value, &secProductInfo)
					if err != nil {
						logs.Error("key [%s], Unmarshal[%s], err:%v ", err)
						getConfSucc = false
						continue
					}
				}
				logs.Debug("get config from etcd, %s %q : %q\n", ev.Type, ev.Kv.Key, ev.Kv.Value)
			}

			if getConfSucc {
				logs.Debug("get config from etcd succ, %v", secProductInfo)
				updateSecproductInfo(secProductInfo)
			}
		}

	}
}
func initSec() (err error) {

	err = initLogs()
	if err != nil {
		logs.Error("init logger failed, err : %v", err)
		return
	}

	err = initRedis()
	if err != nil {
		logs.Error("init redis failed, err : %v", err)
		return
	}
	err = initEtcd()
	if err != nil {
		logs.Error("init etcd failed, err : %v", err)
		return
	}

	err = loadSecConf()
	if err != nil {
		logs.Error("load secconf  failed, err : %v", err)
		return
	}
	service.InitService(secKillConf)
	initSecProductWatcher()
	if err != nil {
		logs.Error("initSecProductWatcher  failed, err : %v", err)
		return
	}
	logs.Info("init seckill success")
	return
}
