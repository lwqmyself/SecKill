package service

import (
	"fmt"
	"github.com/astaxie/beego/logs"
	"time"
)

var (
	secKIllConf *SecKillConf
)

func InitService(serviceConf *SecKillConf) {
	secKIllConf = serviceConf
	logs.Debug("init service success , config : %v", secKIllConf)
}

func SecInfoList() (data []map[string]interface{}, code int, err error) {
	secKIllConf.RwSecProductLock.RLock()

	defer secKIllConf.RwSecProductLock.RUnlock()
	for _, v := range secKIllConf.SecProductInfoMap {

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
	secKIllConf.RwSecProductLock.RLock()
	defer secKIllConf.RwSecProductLock.RUnlock()

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
	secKIllConf.RwSecProductLock.RLock()

	defer secKIllConf.RwSecProductLock.RUnlock()

	v, ok := secKIllConf.SecProductInfoMap[productId]
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
	}

	if now-v.StartTime > 0 {
		start = true

	}
	if now-v.EndTime > 0 {
		start = false
		end = true
		status = "秒杀已经结束啦~(￣▽￣)~*"
	}
	if v.Status == ProductStatusForceSaleOut || v.Status == ProductStatusSaleOut {
		start = false
		end = true
		status = "商品已经售空了(⊙﹏⊙)"
	}
	data = make(map[string]interface{})
	data["product_id"] = productId
	data["start"] = start
	data["end"] = end
	data["status"] = status

	return
}
