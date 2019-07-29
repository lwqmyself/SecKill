package service

import (
	"fmt"
	"github.com/astaxie/beego/logs"
	"time"
)

var (
	secKIllConf *SecKillConf
)

func InitService(serviceConf *SecKillConf)  {
secKIllConf = serviceConf
logs.Debug("init service success , config : %v",secKIllConf)
}

func SecInfo(productId int) (data map[string]interface{},code int,err error) {
	data=make(map[string]interface{},16)
	data["sec"]="hello"
	data["begin"]=time.Now()
	secKIllConf.RwSecProductLock.RLock()

	defer secKIllConf.RwSecProductLock.Unlock()
	v,ok:=secKIllConf.SecProductInfoMap[productId]
	if !ok{
		code= ErrNotFoundProductId
		err = fmt.Errorf("not found product_id : %d",productId)
		return
	}
	data = make(map[string]interface{})
	data["product_id"] = productId
	data["start"] = v.StartTime
	data["end"] = v.EndTime
	data["status"] = v.Status
	return
}