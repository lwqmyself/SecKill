package service

import (
	"fmt"
	"strconv"
	"time"

	"github.com/astaxie/beego/logs"
	"github.com/gomodule/redigo/redis"
)

var (
	secKillConf *SecKillConf
)

func InitService(serviceConf *SecKillConf) (err error) {
	secKillConf = serviceConf

	err = locdBlackList()
	if err != nil {
		logs.Error("init black list err :%v", err)
		return
	}
	logs.Debug("init service success , config : %v", secKillConf)

	err = initProxy2LayerRedis()
	if err != nil {
		logs.Error("load proxy2Layer redis pool failed , err : %v", err)
		return
	}
	err = initLayer2ProxyRedis()
	if err != nil {
		logs.Error("load layer2Proxy redis pool failed , err : %v", err)
		return
	}

	secKillConf.secLimitMgr = &SecLimitMgr{
		UserLimitMap: make(map[int]*Limit, 10000),
		IpLimitMap:   make(map[string]*Limit, 10000),
	}
	secKillConf.SecReqChan = make(chan *SecRequest, secKillConf.SecReqChanSize)
	secKillConf.UserConnMap = make(map[string]chan *SecResult, 10000)
	fmt.Println("开始 initRedisProcessFunc")
	initRedisProcessFunc()
	return
}
func initRedisProcessFunc() {
	for i := 0; i < secKillConf.WriteProxy2LayerGoroutineNum; i++ {
		go WriteHandle()
	}

	for i := 0; i < secKillConf.ReadProxy2LayerGoroutineNum; i++ {
		go ReadHandle()
	}
}

func initProxy2LayerRedis() (err error) {
	fmt.Println(secKillConf.RedisProxy2LayerConf.RedisAddr)
	secKillConf.proxy2LayerRedisPool = &redis.Pool{

		MaxIdle:     secKillConf.RedisProxy2LayerConf.RedisMaxIdle,
		MaxActive:   secKillConf.RedisProxy2LayerConf.RedisMaxActive,
		IdleTimeout: time.Duration(secKillConf.RedisProxy2LayerConf.RedisIdleTimeout) * time.Second,
		Dial: func() (redis.Conn, error) {
			//fmt.Println(secKillConf.RedisProxy2LayerConf.RedisAddr)
			return redis.Dial("tcp", secKillConf.RedisProxy2LayerConf.RedisAddr)
		},
	}

	conn := secKillConf.proxy2LayerRedisPool.Get()
	defer conn.Close()
	_, err = conn.Do("PING")
	if err != nil {
		logs.Error("ping redis failed , err:%v", err)
		return
	}
	return

}

func initLayer2ProxyRedis() (err error) {
	fmt.Println("123" + secKillConf.RedisLayer2ProxyConf.RedisAddr)
	secKillConf.layer2ProxyRedisPool = &redis.Pool{
		MaxIdle:     secKillConf.RedisLayer2ProxyConf.RedisMaxIdle,
		MaxActive:   secKillConf.RedisLayer2ProxyConf.RedisMaxActive,
		IdleTimeout: time.Duration(secKillConf.RedisLayer2ProxyConf.RedisIdleTimeout) * time.Second,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", secKillConf.RedisLayer2ProxyConf.RedisAddr)
		},
	}

	conn := secKillConf.layer2ProxyRedisPool.Get()
	defer conn.Close()

	_, err = conn.Do("ping")
	if err != nil {
		logs.Error("ping redis failed, err:%v", err)
		return
	}

	return
}

func initBlackRedis() (err error) {
	secKillConf.blackRedisPool = &redis.Pool{

		MaxIdle:     secKillConf.RedisBlackConf.RedisMaxIdle,
		MaxActive:   secKillConf.RedisBlackConf.RedisMaxActive,
		IdleTimeout: time.Duration(secKillConf.RedisBlackConf.RedisIdleTimeout) * time.Second,
		Dial: func() (redis.Conn, error) {
			//fmt.Println(secKillConf.RedisBlackConf.RedisAddr)
			return redis.Dial("tcp", secKillConf.RedisBlackConf.RedisAddr)
		},
	}
	//fmt.Println(secKillConf.RedisBlackConf.RedisAddr)
	conn := secKillConf.blackRedisPool.Get()
	defer conn.Close()
	_, err = conn.Do("PING")
	if err != nil {
		logs.Error("ping redis failed , err:%v", err)
		return
	}
	return
}

func locdBlackList() (err error) {
	err = initBlackRedis()
	if err != nil {
		logs.Error("init balck redis failed , err:%v", err)
		return
	}
	conn := secKillConf.blackRedisPool.Get()
	defer conn.Close()
	reply, err := conn.Do("hgetall", "idblacklist")
	idlist, err := redis.Strings(reply, err)
	if err != nil {
		logs.Warn("hget all failed , err : %v", err)
		return
	}

	for _, v := range idlist {
		id, err := strconv.Atoi(v)
		if err != nil {
			logs.Warn("invalid userId [%v]", id)
			continue
		}
		secKillConf.idBlackMap[id] = true
	}

	reply, err = conn.Do("hgetall", "ipblacklist")
	iplist, err := redis.Strings(reply, err)
	if err != nil {
		logs.Warn("hget all failed , err : %v", err)
		return
	}

	for _, v := range iplist {
		secKillConf.ipBlackMap[v] = true
	}
	go SyncIpBlackList()
	go SyncIdBlackList()
	return
}

//同步黑名单
func SyncIpBlackList() {
	var ipList []string
	lastTime := time.Now().Unix()
	for {
		conn := secKillConf.blackRedisPool.Get()
		defer conn.Close()
		reply, err := conn.Do("BLPOP", "blackiplist", time.Second)
		ip, err := redis.String(reply, err)
		if err != nil {
			continue
		}
		curTime := time.Now().Unix()
		ipList = append(ipList, ip)
		if len(ipList) > 100 || curTime-lastTime > 5 {
			secKillConf.RWBlackLock.Lock()
			for _, v := range ipList {
				secKillConf.ipBlackMap[v] = true
			}
			secKillConf.RWBlackLock.Unlock()
			lastTime = curTime
			logs.Info("sync ip list from redis success, ip[%v]", ipList)
		}

	}
}
func SyncIdBlackList() {
	var idList []int
	lastTime := time.Now().Unix()
	for {
		conn := secKillConf.blackRedisPool.Get()
		defer conn.Close()
		reply, err := conn.Do("BLPOP", "blackidlist", time.Second)
		id, err := redis.Int(reply, err)
		if err != nil {
			continue
		}
		curTime := time.Now().Unix()
		idList = append(idList, id)
		if len(idList) > 100 || curTime-lastTime > 5 {
			secKillConf.RWBlackLock.Lock()
			for _, v := range idList {
				secKillConf.idBlackMap[v] = true
			}
			secKillConf.RWBlackLock.Unlock()
			lastTime = curTime
			logs.Info("sync id list from redis success, id[%v]", idList)
		}
	}
}
