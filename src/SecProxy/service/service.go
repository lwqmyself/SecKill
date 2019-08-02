package service

import (
	"crypto/md5"
	"fmt"
	"github.com/astaxie/beego/logs"
	"github.com/gomodule/redigo/redis"
	"strconv"
	"time"
)

var (
	secKillConf *SecKillConf
)

func InitService(serviceConf *SecKillConf) {
	secKillConf = serviceConf

	locdBlackList()

	logs.Debug("init service success , config : %v", secKillConf)
}

func initBlackRedis() (err error) {
	secKillConf.blackRedisPool = &redis.Pool{

		MaxIdle:     secKillConf.RedisBlackConf.RedisMaxIdle,
		MaxActive:   secKillConf.RedisBlackConf.RedisMaxActive,
		IdleTimeout: time.Duration(secKillConf.RedisBlackConf.RedisIdleTimeout) * time.Second,
		Dial: func() (redis.Conn, error) {
			fmt.Println(secKillConf.RedisBlackConf.RedisAddr)
			return redis.Dial("tcp", secKillConf.RedisBlackConf.RedisAddr)
		},
	}
	fmt.Println(secKillConf.RedisBlackConf.RedisAddr)
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
		secKillConf.idBlackMap[v] = true
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

func SecInfoList() (data []map[string]interface{}, code int, err error) {
	secKillConf.RWSecProductLock.RLock()

	defer secKillConf.RWSecProductLock.RUnlock()
	for _, v := range secKillConf.SecProductInfoMap {

		item, _, err := SecInfoById(v.ProductId)
		if err != nil {
			logs.Error("get product_id[%d] failed , err : %v", err)
			continue
		}
		logs.Debug("get product[%d], result[%v] map[%v] ", v.ProductId, item, secKIllConf.SecProductInfoMap)
		data = append(data, item)
	}
	return
}

func SecInfo(productId int) (data []map[string]interface{}, code int, err error) {

	//写锁
	secKillConf.RWSecProductLock.RLock()
	defer secKillConf.RWSecProductLock.RUnlock()

	item, code, err := SecInfoById(productId)
	if err != nil {
		return
	}
	data = append(data, item)

	return
}
func SecInfoById(productId int) (data map[string]interface{}, code int, err error) {

	/*data["sec"]="hello"
	data["begin"]=time.Now()*/
	secKillConf.RWSecProductLock.RLock()

	defer secKillConf.RWSecProductLock.RUnlock()

	v, ok := secKillConf.SecProductInfoMap[productId]
	if !ok {
		code = ErrNotFoundProductId
		err = fmt.Errorf("not found product_id : %d", productId)
		return
	}

	start := false
	end := true
	status := "success"

	now := time.Now().Unix()

	if now-v.StartTime < 0 {
		start = false
		end = false
		status = "秒杀还未开始哦O(∩_∩)O"
		code = ErrActiveNotStart
	}

	if now-v.StartTime > 0 {
		start = true

	}
	if now-v.EndTime > 0 {
		start = false
		end = true
		status = "秒杀已经结束啦~(￣▽￣)~*"
		code = ErrActiveAlreadyEnd
	}
	if v.Status == ProductStatusForceSaleOut || v.Status == ProductStatusSaleOut {
		start = false
		end = true
		status = "商品已经售空了(⊙﹏⊙)"
		code = ErrActiveSaleOut
	}
	data = make(map[string]interface{})
	data["product_id"] = productId
	data["start"] = start
	data["end"] = end
	data["status"] = status

	return
}

func UserCheck(req *SecRequest) (err error) {
	//白名单检测
	found := false
	for _, refer := range secKillConf.ReferWhiteList {
		if refer == req.ClientRefence {
			found = true
			break
		}
	}

	if !found {
		err = fmt.Errorf("invalid request")
		logs.Warn("user[%d] is reject by refer , req[%v]", req.UserId, req)
		return
	}
	authData := fmt.Sprintf("%d:%s", req.UserId, secKillConf.CookieSecretKey)
	authSign := fmt.Sprintf("%x", md5.Sum([]byte(authData)))
	if authSign != req.UserAuthSign {
		err = fmt.Errorf("invalid user cookie auth")
		return
	}
	return
}

func SecKill(req *SecRequest) (data map[string]interface{}, code int, err error) {

	secKillConf.RWSecProductLock.RLock()
	defer secKillConf.RWSecProductLock.RUnlock()
	err = UserCheck(req)
	if err != nil {
		code = ErrUserCheckAuthFailed
		fmt.Println(code)
		logs.Warn("userId [%d] invalid , check failed , req[%v]", req.UserId, req)
		return
	}

	//处理多次请求，防止机器人
	err = antiSpam(req)
	if err != nil {
		code = ErrUserServiceBusy
		logs.Warn("userId[%d] invalid , check failed , req[%v]", req.UserId, req)
	}
	data, code, err := SecInfoById(req.ProductId)
	if err != nil {
		logs.Warn("userId[%d] secInfoBy Id failed , req[%v]", req.UserId, req)

	}
	if code != 0 {
		logs.Warn("userId[%d] secInfoById failed,code[%d] req[%v]", req.UserId, code, req)
	}

	return
}
