package hitball

import (
	"encoding/json"
	"math/rand"
	"time"

	"github.com/shangzongyu/mqant/conf"
	"github.com/shangzongyu/mqant/gate"
	"github.com/shangzongyu/mqant/log"
	"github.com/shangzongyu/mqant/module"
	"github.com/shangzongyu/mqant/module/base"
)

var Module = func() module.Module {
	gate := new(Hitball)
	return gate
}

// 一定要记得在confin.json配置这个模块的参数,否则无法使用
type Hitball struct {
	basemodule.BaseModule
	room    *Room
	proTime int64
	table   *Table
}

// GetRandomString 生成随机字符串
func GetRandomString(lenght int) string {
	str := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	bytes := []byte(str)
	result := []byte{}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < lenght; i++ {
		result = append(result, bytes[r.Intn(len(bytes))])
	}
	return string(result)
}

func (hb *Hitball) GetType() string {
	//很关键,需要与配置文件中的Module配置对应
	return "Hitball"
}

func (hb *Hitball) Version() string {
	//可以在监控时了解代码版本
	return "1.0.0"
}

func (hb *Hitball) OnInit(app module.App, settings *conf.ModuleSettings) {
	hb.BaseModule.OnInit(hb, app, settings)
	hb.room = NewRoom(hb)
	hb.table, _ = hb.room.GetEmptyTable()
	//hb.SetListener(new(chat.Listener))
	hb.GetServer().RegisterGO("HD_Move", hb.move)
	hb.GetServer().RegisterGO("HD_Join", hb.join)
	hb.GetServer().RegisterGO("HD_Fire", hb.fire)
	hb.GetServer().RegisterGO("HD_EatCoin", hb.eatCoin)
}

func (hb *Hitball) Run(closeSig chan bool) {
	hb.table.Start()
}

func (hb *Hitball) OnDestroy() {
	//一定别忘了关闭RPC
	hb.GetServer().OnDestroy()
}

func (hb *Hitball) join(session gate.Session, msg map[string]interface{}) (result string, err string) {
	if session.GetUserId() == "" {
		session.Bind(GetRandomString(8))
		//return "","no login"
	}
	erro := hb.table.PutQueue("Join", session)
	if erro != nil {
		return "", erro.Error()
	}
	return "success", ""
}

func (hb *Hitball) fire(session gate.Session, msg map[string]interface{}) (result string, err string) {
	if msg["Angle"] == nil || msg["Power"] == nil || msg["X"] == nil || msg["Y"] == nil {
		err = "Angle , Power X ,Y cannot be nil"
		return
	}
	Angle := msg["Angle"].(float64)
	Power := msg["Power"].(float64)
	X := msg["X"].(float64)
	Y := msg["Y"].(float64)
	erro := hb.table.PutQueue("Fire", session, float64(X), float64(Y), float64(Angle), float64(Power))
	if erro != nil {
		return "", erro.Error()
	}
	return "success", ""
}

func (hb *Hitball) eatCoin(session gate.Session, msg map[string]interface{}) (result string, err string) {
	if msg["Id"] == nil {
		err = "Id cannot be nil"
		return
	}
	Id := int(msg["Id"].(float64))
	erro := hb.table.PutQueue("EatCoins", session, Id)
	if erro != nil {
		return "", erro.Error()
	}
	return "success", ""
}

func (hb *Hitball) move(session gate.Session, msg map[string]interface{}) (result string, err string) {
	if msg["war"] == nil || msg["wid"] == nil || msg["x"] == nil || msg["y"] == nil {
		err = "war , wid ,x ,y cannot be nil"
		return
	}
	//log.Debug("exct time %d", (time.Now().UnixNano()-hb.proTime)/1000000)
	//hb.proTime = time.Now().UnixNano()
	//war := msg["war"].(string)
	//wid := msg["wid"].(string)
	x := msg["x"].(float64)
	y := msg["y"].(float64)
	//passWord:=msg["passWord"].(string)
	roles := []map[string]float64{
		map[string]float64{
			"x": x,
			"y": y,
		},
	}
	re := map[string]interface{}{}
	re["roles"] = roles
	b, _ := json.Marshal(re)
	e := session.SendNR("Hitball/OnMove", b)
	if e != "" {
		log.Error(e)
	}
	//log.Debug(fmt.Sprintf("move success x:%v,y:%v", x, y))
	return "success", ""
}
