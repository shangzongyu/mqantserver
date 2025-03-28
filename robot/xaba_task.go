// Copyright 2014 hey Author. All Rights Reserved.
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

package main

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/shangzongyu/armyant/task"
	"github.com/shangzongyu/mqantserver/robot/xaba"
	xaba_task "github.com/shangzongyu/mqantserver/robot/xaba"
)

func main() {

	task := task.LoopTask{
		C: 100, //并发数 两人1桌 建立两张桌子
	}
	manager := xaba_task.NewManager(task)
	fmt.Println("开始压测请等待")
	task.Run(manager)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	task.Stop()
	os.Exit(1)
}
