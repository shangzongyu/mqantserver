// Copyright 2014 loolgame Author. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// 与客户端通信的自定义粘包示例，需要mqant v1.6.4版本以上才能运行
// 该示例只用于简单的演示，并没有实现具体的粘包协议

package mgate

import (
	"bufio"
	"time"

	"github.com/shangzongyu/mqant/gate"
	"github.com/shangzongyu/mqant/log"
	"github.com/shangzongyu/mqant/module"
	"github.com/shangzongyu/mqant/network"
)

func NewAgent(module module.RPCModule) *CustomAgent {
	a := &CustomAgent{
		module: module,
	}
	return a
}

type CustomAgent struct {
	gate.Agent
	module                       module.RPCModule
	session                      gate.Session
	conn                         network.Conn
	r                            *bufio.Reader
	w                            *bufio.Writer
	gate                         gate.Gate
	revNum                       int64
	sendNum                      int64
	lastStorageHeartbeatDataTime time.Duration //上一次发送存储心跳时间
	isclose                      bool
}

func (ca *CustomAgent) OnInit(gate gate.Gate, conn network.Conn) error {
	log.Info("CustomAgent", "OnInit")
	ca.conn = conn
	ca.gate = gate
	ca.r = bufio.NewReader(conn)
	ca.w = bufio.NewWriter(conn)
	ca.isclose = false
	ca.revNum = 0
	ca.sendNum = 0
	return nil
}

// WriteMsg 给客户端发送消息
func (ca *CustomAgent) WriteMsg(topic string, body []byte) error {
	ca.sendNum++
	//粘包完成后调下面的语句发送数据
	//ca.w.Write()
	return nil
}

func (ca *CustomAgent) Run() (err error) {
	log.Info("CustomAgent", "开始读数据了")

	ca.session, err = ca.gate.NewSessionByMap(map[string]interface{}{
		"Sessionid": "生成一个随机数",
		"Network":   ca.conn.RemoteAddr().Network(),
		"IP":        ca.conn.RemoteAddr().String(),
		"Serverid":  ca.module.GetServerId(),
		"Settings":  make(map[string]string),
	})

	//这里可以循环读取客户端的数据

	//这个函数返回后连接就会被关闭
	return nil
}

// OnRecover 接收到一个数据包
func (ca *CustomAgent) OnRecover(topic string, msg []byte) {
	//通过解析的数据得到
	moduleType := ""
	_func := ""

	//如果要对这个请求进行分布式跟踪调试,就执行下面这行语句
	//a.session.CreateRootSpan("gate")

	//然后请求后端模块，第一个参数为session
	result, e := ca.module.RpcInvoke(moduleType, _func, ca.session, msg)
	log.Info("result", result)
	log.Info("error", e)

	//回复客户端
	ca.WriteMsg(topic, []byte("请求成功了谢谢"))

	ca.heartbeat()
}

func (ca *CustomAgent) heartbeat() {
	//自定义网关需要你自己设计心跳协议
	if ca.GetSession().GetUserId() != "" {
		//这个链接已经绑定Userid
		interval := int64(ca.lastStorageHeartbeatDataTime) + int64(ca.gate.Options().Heartbeat) //单位纳秒
		if interval < time.Now().UnixNano() {
			//如果用户信息存储心跳包的时长已经大于一秒
			if ca.gate.GetStorageHandler() != nil {
				ca.gate.GetStorageHandler().Heartbeat(ca.GetSession())
				ca.lastStorageHeartbeatDataTime = time.Duration(time.Now().UnixNano())
			}
		}
	}
}

func (ca *CustomAgent) Close() {
	log.Info("CustomAgent", "主动断开连接")
	ca.conn.Close()
}

func (ca *CustomAgent) OnClose() error {
	ca.isclose = true
	log.Info("CustomAgent", "连接断开事件")
	//这个一定要调用，不然gate可能注销不了,造成内存溢出
	ca.gate.GetAgentLearner().DisConnect(this) //发送连接断开的事件
	return nil
}

func (ca *CustomAgent) Destroy() {
	ca.conn.Destroy()
}

func (ca *CustomAgent) RevNum() int64 {
	return ca.revNum
}

func (ca *CustomAgent) SendNum() int64 {
	return ca.sendNum
}

func (ca *CustomAgent) IsClosed() bool {
	return ca.isclose
}

func (ca *CustomAgent) GetSession() gate.Session {
	return ca.session
}
