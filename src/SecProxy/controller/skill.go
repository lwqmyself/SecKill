package controller

import (
	"SecProxy/service"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
)

type SkillController struct {
	beego.Controller
}

func (p *SkillController) SecKill() {
p.Data["json"] = "sec kill"
p.ServeJSON()

}
func (p *SkillController) SecInfo() {
	productId,err:= p.GetInt("product_id")
	result := make(map[string]interface{})
	result["code"] = 0
	result["message"] = "success"
	defer func() {
		p.Data["json"] = result
		p.ServeJSON()
	}()
	if err!=nil{

		result["code"] = 1001
		result["message"] = "invalid product_id"
		logs.Error("invalid request get product_id failed , error : %v",err)
		return
	}
	data,code,err:=service.SecInfo(productId)
	if err != nil {
		result["code"] = code
		result["message"] = "success"
		logs.Error("invalid request get product_id failed , error : %v",err)
		return
	}
	result["data"] = data

}
