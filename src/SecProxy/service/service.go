package service

import (
	"crypto/md5"
	"fmt"
	"github.com/astaxie/beego/logs"

	"time"
)

func NewSecRequest() (secRequest *SecRequest) {
	secRequest = &SecRequest{
		ResultChan: make(chan *SecResult, 1),
	}

	return
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
		logs.Debug("get product[%d], result[%v] map[%v] ", v.ProductId, item, secKillConf.SecProductInfoMap)
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
		end = false

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
	/*err = UserCheck(req)
	if err != nil {
		code = ErrUserCheckAuthFailed
		fmt.Println(code)
		logs.Warn("userId [%d] invalid , check failed , req[%v]", req.UserId, req)
		return
	}*/

	//处理多次请求，防止机器人

	err = antiSpam(req)
	if err != nil {
		code = ErrUserServiceBusy
		logs.Warn("userId[%d] invalid , check failed , req[%v]", req.UserId, req)
		return
	}
	data, code, err = SecInfoById(req.ProductId)
	logs.Debug("code : %d", code)
	if err != nil {
		logs.Warn("userId[%d] secInfoBy Id failed , req[%v]", req.UserId, req)
		return

	}
	if code != 0 {
		logs.Warn("userId[%d] secInfoById failed,code[%d] req[%v]", req.UserId, code, req)
		return
	}

	userKey := fmt.Sprintf("%s_%s", req.UserId, req.ProductId)
	secKillConf.UserConnMap[userKey] = req.ResultChan
	logs.Debug(secKillConf.UserConnMap)
	secKillConf.SecReqChan <- req

	ticker := time.NewTicker(time.Second * 10)

	defer func() {
		ticker.Stop()
		secKillConf.UserConnMapLock.Lock()
		delete(secKillConf.UserConnMap, userKey)
		secKillConf.UserConnMapLock.Unlock()
	}()

	select {
	case <-ticker.C:
		code = ErrProcessTimeout
		err = fmt.Errorf("request timeout")

		return
	case <-req.CloseNotify:
		code = ErrClientClosed
		err = fmt.Errorf("client already closed")
		return
	case result := <-req.ResultChan:
		code = result.Code

		data["product_id"] = result.ProductId
		data["token"] = result.Token
		data["user_id"] = result.UserId
		logs.Debug(data)
		logs.Debug(code)
		return
	}

	return
}
