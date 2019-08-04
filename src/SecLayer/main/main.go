package main

import (
	"SecLayer/service"
	"fmt"
	"github.com/astaxie/beego/logs"
)

func main() {
	//1.加载配置文件
	err := initConfig("ini", "./conf/seclayer.conf")
	if err != nil {
		logs.Debug("init config failed , err : %v", err)
		panic(fmt.Sprintf("init config failed , err : %v", err))
	}
	logs.Debug("load config succ, data:%v", appConfig)
	//2.初始化日志库
	err = initLogger()
	if err != nil {
		logs.Error("init logger failed , err : %v", err)
		panic(fmt.Sprintf("init logger failed , err : %v", err))
	}
	logs.Debug("init logger success")
	//3.初始化秒杀逻辑
	err = service.InitSecLayer(appConfig)
	if err != nil {
		msg := fmt.Sprintf("init SecKill failed , err : %v", err)
		logs.Error(msg)
		panic(msg)
	}
	logs.Debug("init sec layer success")
	//4.运行业务逻辑
	err = service.Run()
	if err != nil {
		logs.Error("service run returned , err : %v", err)
	}
	logs.Info("service run exited")
}
