package main

import (
	_ "SecAdmin/router"
	"fmt"
	"github.com/astaxie/beego"
)

func main() {
	err := initAll()
	if err != nil {
		panic(fmt.Sprintf("init database failed, err:%v", err))
		return
	}
	beego.Run()
}
