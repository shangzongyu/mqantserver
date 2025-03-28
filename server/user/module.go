package user

import (
	"fmt"

	"github.com/shangzongyu/mqant/conf"
	"github.com/shangzongyu/mqant/module"
	"github.com/shangzongyu/mqant/module/base"
)

// 一定要记得在confin.json配置这个模块的参数,否则无法使用

var Module = func() module.Module {
	user := new(User)
	return user
}

type User struct {
	basemodule.BaseModule
}

func (u *User) GetType() string {
	//很关键,需要与配置文件中的Module配置对应
	return "User"
}

func (u *User) Version() string {
	//可以在监控时了解代码版本
	return "1.0.0"
}

func (u *User) OnInit(app module.App, settings *conf.ModuleSettings) {
	u.BaseModule.OnInit(u, app, settings)

	u.GetServer().RegisterGO("mongodb", u.mongodb) //演示后台模块间的rpc调用
}

func (u *User) Run(closeSig chan bool) {
}

func (u *User) OnDestroy() {
	//一定别忘了关闭RPC
	u.GetServer().OnDestroy()
}

func (u *User) mongodb() (string, string) {

	return fmt.Sprintf("My is Login Module"), ""
}
