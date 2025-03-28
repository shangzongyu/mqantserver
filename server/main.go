package main

import (
	"fmt"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"sync"

	"github.com/nats-io/nats.go"
	"github.com/shangzongyu/mqant"
	"github.com/shangzongyu/mqant/gate"
	"github.com/shangzongyu/mqant/module"
	"github.com/shangzongyu/mqant/module/modules"
	"github.com/shangzongyu/mqant/registry"
	"github.com/shangzongyu/mqant/selector"
	"github.com/shangzongyu/mqantserver/server/chat"
	"github.com/shangzongyu/mqantserver/server/gate"
	"github.com/shangzongyu/mqantserver/server/helloworld"
	"github.com/shangzongyu/mqantserver/server/hitball"
	"github.com/shangzongyu/mqantserver/server/login"
	"github.com/shangzongyu/mqantserver/server/user"
	"github.com/shangzongyu/mqantserver/server/xaxb"
	"github.com/shangzongyu/mqantserver/webapp"
)

func main() {
	go func() {
		http.ListenAndServe("0.0.0.0:6060", nil)
	}()
	rs := registry.DefaultRegistry //etcdv3.NewRegistry()
	nc, err := nats.Connect(nats.DefaultURL, nats.MaxReconnects(10000))
	if err != nil {

	}
	app := mqant.CreateApp(
		module.Debug(true), //只有是在调试模式下才会在控制台打印日志, 非调试模式下只在日志文件中输出日志
		module.Nats(nc),
		module.Registry(rs),
		module.SetJudgeGuest(func(session gate.Session) bool {
			if session.GetUserId() == "" {
				return true
			} else {
				return false
			}
		}),
	)
	app.Options().Selector.Init(selector.SetStrategy(func(services []*registry.Service) selector.Next {
		var nodes []*registry.Node

		// Filter the nodes for datacenter
		for _, service := range services {
			for _, node := range service.Nodes {
				nodes = append(nodes, node)
				//if node.Metadata["type"] == "helloworld" {
				//	nodes = append(nodes, node)
				//}
			}
		}

		var mtx sync.Mutex
		//log.Info("services[0] $v",services[0].Nodes[0])
		return func() (*registry.Node, error) {
			mtx.Lock()
			defer mtx.Unlock()
			if len(nodes) == 0 {
				return nil, fmt.Errorf("no node")
			}
			index := rand.Intn(int(len(nodes)))
			return nodes[index], nil
		}
	}))
	//app.Route("Chat",ChatRoute)
	app.Run( //只有是在调试模式下才会在控制台打印日志, 非调试模式下只在日志文件中输出日志
		modules.MasterModule(),
		hitball.Module(),
		mgate.Module(), //这是默认网关模块,是必须的支持 TCP,websocket,MQTT协议
		helloworld.Module(),
		login.Module(), //这是用户登录验证模块
		chat.Module(),
		user.Module(),
		webapp.Module(),
		xaxb.Module(),
		//tracing.Module(), //很多初学者不会改文件路径，先移除了
	) //这是聊天模块

}
