package main

import (
	_ "SecProxy/router"
	"fmt"
	"github.com/astaxie/beego"
	_"github.com/astaxie/beego/logs"
)


func main() {
	fmt.Println("开始")
	err := initConfig()
	if err != nil {
		panic(err)
		return
	}
	err = initSec()
	if err != nil {
		panic(err)
		return
	}
	beego.Run()
}
