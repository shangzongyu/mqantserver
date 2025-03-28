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

package hitball

import (
	"fmt"
	"reflect"
	"runtime"
	"sync"

	"github.com/shangzongyu/mqant/log"
	"github.com/yireyun/go-queue"
)

type QueueMsg struct {
	Func   string
	Params []interface{}
}
type QueueReceive interface {
	Receive(msg *QueueMsg, index int)
}
type BaseTable struct {
	functions     map[string]interface{}
	receive       QueueReceive
	queue0        *queue.EsQueue
	queue1        *queue.EsQueue
	currentWQueue int //当前写的队列
	lock          *sync.RWMutex
}

func (bat *BaseTable) Init() {
	bat.functions = map[string]interface{}{}
	bat.queue0 = queue.NewQueue(256)
	bat.queue1 = queue.NewQueue(256)
	bat.currentWQueue = 0
	bat.lock = new(sync.RWMutex)
}

func (bat *BaseTable) SetReceive(receive QueueReceive) {
	bat.receive = receive
}
func (bat *BaseTable) Register(id string, f interface{}) {

	if _, ok := bat.functions[id]; ok {
		panic(fmt.Sprintf("function id %v: already registered", id))
	}

	bat.functions[id] = f
}

// PutQueue 协成安全,任意协成可调用
func (bat *BaseTable) PutQueue(_func string, params ...interface{}) error {
	ok, quantity := bat.wqueue().Put(&QueueMsg{
		Func:   _func,
		Params: params,
	})
	if !ok {
		return fmt.Errorf("Put Fail, quantity:%v\n", quantity)
	} else {
		return nil
	}

}

// switchqueue 切换并且返回读的队列
func (bat *BaseTable) switchqueue() *queue.EsQueue {
	bat.lock.Lock()
	if bat.currentWQueue == 0 {
		bat.currentWQueue = 1
		bat.lock.Unlock()
		return bat.queue0
	} else {
		bat.currentWQueue = 0
		bat.lock.Unlock()
		return bat.queue1
	}

}
func (bat *BaseTable) wqueue() *queue.EsQueue {
	bat.lock.Lock()
	if bat.currentWQueue == 0 {
		bat.lock.Unlock()
		return bat.queue0
	} else {
		bat.lock.Unlock()
		return bat.queue1
	}

}

// ExecuteEvent 【每帧调用】执行队列中的所有事件
func (bat *BaseTable) ExecuteEvent(arge interface{}) {
	ok := true
	queue := bat.switchqueue()
	index := 0
	for ok {
		val, _ok, _ := queue.Get()
		index++
		if _ok {
			if bat.receive != nil {
				bat.receive.Receive(val.(*QueueMsg), index)
			} else {
				msg := val.(*QueueMsg)
				function, ok := bat.functions[msg.Func]
				if !ok {
					fmt.Println(fmt.Sprintf("Remote function(%s) not found", msg.Func))
					continue
				}
				f := reflect.ValueOf(function)
				in := make([]reflect.Value, len(msg.Params))
				for k, _ := range in {
					in[k] = reflect.ValueOf(msg.Params[k])
				}
				_runFunc := func() {
					defer func() {
						if r := recover(); r != nil {
							var rn = ""
							switch r.(type) {

							case string:
								rn = r.(string)
							case error:
								rn = r.(error).Error()
							}
							buf := make([]byte, 1024)
							l := runtime.Stack(buf, false)
							errstr := string(buf[:l])
							log.Error("table qeueu event(%s) exec fail error:%s \n ----Stack----\n %s", msg.Func, rn, errstr)
						}
					}()
					f.Call(in)
				}
				_runFunc()
			}
		}
		ok = _ok
	}
}
