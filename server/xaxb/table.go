// Copyright 2014 mqantserver Author. All Rights Reserved.
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

package xaxb

import (
	"container/list"
	"math/rand"
	"sync"
	"time"

	"github.com/shangzongyu/mqant-modules/room"
	"github.com/shangzongyu/mqant/gate"
	"github.com/shangzongyu/mqant/log"
	"github.com/shangzongyu/mqant/module"
	"github.com/shangzongyu/mqant/module/modules/timer"
	"github.com/shangzongyu/mqantserver/server/xaxb/objects"
)

func RandInt64(min, max int64) int64 {
	if min >= max {
		return max
	}

	return rand.Int63n(max-min) + min
}

//游戏逻辑相关的代码
//第一阶段 空档期，等待押注
//第二阶段 押注期  可以押注
//第三阶段 开奖期  开奖
//第四阶段 结算期  结算

type Table struct {
	fsm FSM
	room.BaseTableImp
	room.QueueTable
	room.UnifiedSendMessageTable
	room.TimeOutTable
	module                  module.RPCModule
	seats                   []*objects.Player
	viewer                  *list.List //观众
	seatMax                 int        //房间最大座位数
	currentId               int64
	currentFrame            int64 //当前帧
	syncFrame               int64 //上一次同步数据的帧
	stoped                  bool
	writeLock               sync.Mutex
	VoidPeriodHandler       FSMHandler
	IdlePeriodHandler       FSMHandler
	BettingPeriodHandler    FSMHandler
	OpeningPeriodHandler    FSMHandler
	SettlementPeriodHandler FSMHandler
	step1                   int64 //空档期帧frame
	step2                   int64 //押注期帧frame
	step3                   int64 //开奖期帧frame
	step4                   int64 //结算期帧frame
}

func NewTable(module module.RPCModule, tableId int) *Table {
	this := &Table{
		module:       module,
		stoped:       true,
		seatMax:      2,
		currentId:    0,
		currentFrame: 0,
		syncFrame:    0,
	}
	this.BaseTableImpInit(tableId, this)
	this.QueueInit()
	this.UnifiedSendMessageTableInit(this)
	this.TimeOutTableInit(this, this, 60)
	//游戏逻辑状态机
	this.InitFsm()
	this.seats = make([]*objects.Player, this.seatMax)
	this.viewer = list.New()

	this.Register("SitDown", this.SitDown)
	this.Register("GetLevel", this.getLevel)
	this.Register("StartGame", this.StartGame)
	this.Register("PauseGame", this.PauseGame)
	this.Register("Stake", this.Stake)
	for indexSeat, _ := range this.seats {
		this.seats[indexSeat] = objects.NewPlayer(indexSeat)
	}

	return this
}
func (t *Table) GetModule() module.RPCModule {
	return t.module
}
func (t *Table) GetSeats() []room.BasePlayer {
	m := make([]room.BasePlayer, len(t.seats))
	for i, seat := range t.seats {
		m[i] = seat
	}
	return m
}
func (t *Table) GetViewer() *list.List {
	return t.viewer
}

// OnNetBroken 玩家断线,游戏暂停
func (t *Table) OnNetBroken(player room.BasePlayer) {
	player.OnNetBroken()
}

// VerifyAccessAuthority 访问权限校验
func (t *Table) VerifyAccessAuthority(userId string, bigRoomId string) bool {
	_, tableId, transactionId, err := room.ParseBigRoomId(bigRoomId)
	if err != nil {
		log.Error(err.Error())
		return false
	}
	if (tableId != t.TableId()) || (transactionId != t.TransactionId()) {
		log.Error("transactionId!=this.TransactionId()", transactionId, t.TransactionId())
		return false
	}
	return true
}

func (t *Table) AllowJoin() bool {
	t.writeLock.Lock()
	ready := true
	if t.currentId > 2 {
		t.writeLock.Unlock()
		return false
	}
	t.currentId++
	t.writeLock.Unlock()
	return true
	//for _, seat := range this.GetSeats() {
	//	if seat.Bind() == false {
	//		//还没有准备好
	//		ready = false
	//		break
	//	}
	//}

	return !ready
}

func (t *Table) OnCreate() {
	t.BaseTableImp.OnCreate()
	t.ResetTimeOut()
	log.Debug("Table", "OnCreate")
	if t.stoped {
		t.stoped = false
		timewheel.GetTimeWheel().AddTimer(1000*time.Millisecond, nil, t.Update)
		//go func() {
		//	//这里设置为500ms
		//	tick := time.NewTicker(1000 * time.Millisecond)
		//	defer func() {
		//		tick.Stop()
		//	}()
		//	for !this.stoped {
		//		select {
		//		case <-tick.C:
		//			this.Update(nil)
		//		}
		//	}
		//}()
	}
}
func (t *Table) OnStart() {
	log.Debug("Table", "OnStart")
	for _, player := range t.seats {
		player.Coin = 100000
		player.Weight = 0
		player.Target = 0
		player.Stake = false
	}
	//将游戏状态设置到空闲期
	t.fsm.Call(IdlePeriodEvent)
	t.step1 = 0
	t.step2 = 0
	t.step3 = 0
	t.step4 = 0
	t.currentFrame = 0
	t.syncFrame = 0
	t.BaseTableImp.OnStart()
}

func (t *Table) OnResume() {
	t.BaseTableImp.OnResume()
	log.Debug("Table", "OnResume")
	t.NotifyResume()
}

func (t *Table) OnPause() {
	t.BaseTableImp.OnPause()
	log.Debug("Table", "OnPause")
	t.NotifyPause()
}

func (t *Table) OnStop() {
	t.BaseTableImp.OnStop()
	log.Debug("Table", "OnStop")
	//将游戏状态设置到空档期
	t.fsm.Call(VoidPeriodEvent)
	t.NotifyStop()
	t.ExecuteCallBackMsg() //统一发送数据到客户端
	for _, player := range t.seats {
		player.OnUnBind()
	}

	var nv *list.Element
	for e := t.viewer.Front(); e != nil; e = nv {
		nv = e.Next()
		t.viewer.Remove(e)
	}
}
func (t *Table) OnDestroy() {
	t.BaseTableImp.OnDestroy()
	log.Debug("BaseTableImp", "OnDestroy")
	t.stoped = true
}

func (t *Table) onGameOver() {
	t.Finish()
}

// Update 牌桌主循环 , 定帧计算所有玩家的位置
func (t *Table) Update(arge interface{}) {
	t.ExecuteEvent(arge) //执行这一帧客户端发送过来的消息
	if t.State() == room.Active {
		t.currentFrame++

		if t.currentFrame-t.syncFrame >= 1 {
			//每帧同步一次
			t.syncFrame = t.currentFrame
			t.StateSwitch()
		}

		ready := true
		for _, seat := range t.GetSeats() {
			if seat.Bind() == false {
				//还没有准备好
				ready = false
				break
			}
		}
		if ready == false {
			//有玩家离开了牌桌,牌桌退出
			t.Finish()
		}

	} else if t.State() == room.Initialized {
		ready := true
		for _, seat := range t.GetSeats() {
			if seat.SitDown() == false {
				//还没有准备好
				ready = false
				break
			}
		}
		if ready {
			t.Start() //开始游戏了
		}
	}

	t.ExecuteCallBackMsg() //统一发送数据到客户端
	t.CheckTimeOut()
	if !t.stoped {
		timewheel.GetTimeWheel().AddTimer(100*time.Millisecond, nil, t.Update)
	}
}

func (t *Table) Exit(session gate.Session) error {
	player := t.GetBindPlayer(session)
	if player != nil {
		playerImp := player.(*objects.Player)
		playerImp.OnUnBind()
		return nil
	}
	return nil
}

func (t *Table) getSeatsMap() []map[string]interface{} {
	m := make([]map[string]interface{}, len(t.seats))
	for i, player := range t.seats {
		if player != nil {
			m[i] = player.SerializableMap()
		}
	}
	return m
}

// getLevel 玩家获取关卡信息
func (t *Table) getLevel(session gate.Session) {
	playerImp := t.GetBindPlayer(session)
	if playerImp != nil {
		player := playerImp.(*objects.Player)
		player.OnRequest(session)
		player.OnSitDown()
	}
}
