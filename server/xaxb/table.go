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
	"github.com/shangzongyu/mqant-modules/room"
	"github.com/shangzongyu/mqant/gate"
	"github.com/shangzongyu/mqant/log"
	"github.com/shangzongyu/mqant/module"
	"github.com/shangzongyu/mqant/module/modules/timer"
	"github.com/shangzongyu/mqantserver/server/xaxb/objects"
	"math/rand"
	"sync"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

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
	current_id              int64
	current_frame           int64 //当前帧
	sync_frame              int64 //上一次同步数据的帧
	stoped                  bool
	writelock               sync.Mutex
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
		module:        module,
		stoped:        true,
		seatMax:       2,
		current_id:    0,
		current_frame: 0,
		sync_frame:    0,
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
func (self *Table) GetModule() module.RPCModule {
	return self.module
}
func (self *Table) GetSeats() []room.BasePlayer {
	m := make([]room.BasePlayer, len(self.seats))
	for i, seat := range self.seats {
		m[i] = seat
	}
	return m
}
func (self *Table) GetViewer() *list.List {
	return self.viewer
}

/*
*
玩家断线,游戏暂停
*/
func (self *Table) OnNetBroken(player room.BasePlayer) {
	player.OnNetBroken()
}

// //访问权限校验
func (self *Table) VerifyAccessAuthority(userId string, bigRoomId string) bool {
	_, tableid, transactionId, err := room.ParseBigRoomId(bigRoomId)
	if err != nil {
		log.Error(err.Error())
		return false
	}
	if (tableid != self.TableId()) || (transactionId != self.TransactionId()) {
		log.Error("transactionId!=this.TransactionId()", transactionId, self.TransactionId())
		return false
	}
	return true
}
func (self *Table) AllowJoin() bool {
	self.writelock.Lock()
	ready := true
	if self.current_id > 2 {
		self.writelock.Unlock()
		return false
	}
	self.current_id++
	self.writelock.Unlock()
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
func (self *Table) OnCreate() {
	self.BaseTableImp.OnCreate()
	self.ResetTimeOut()
	log.Debug("Table", "OnCreate")
	if self.stoped {
		self.stoped = false
		timewheel.GetTimeWheel().AddTimer(1000*time.Millisecond, nil, self.Update)
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
func (self *Table) OnStart() {
	log.Debug("Table", "OnStart")
	for _, player := range self.seats {
		player.Coin = 100000
		player.Weight = 0
		player.Target = 0
		player.Stake = false
	}
	//将游戏状态设置到空闲期
	self.fsm.Call(IdlePeriodEvent)
	self.step1 = 0
	self.step2 = 0
	self.step3 = 0
	self.step4 = 0
	self.current_frame = 0
	self.sync_frame = 0
	self.BaseTableImp.OnStart()
}
func (self *Table) OnResume() {
	self.BaseTableImp.OnResume()
	log.Debug("Table", "OnResume")
	self.NotifyResume()
}
func (self *Table) OnPause() {
	self.BaseTableImp.OnPause()
	log.Debug("Table", "OnPause")
	self.NotifyPause()
}
func (self *Table) OnStop() {
	self.BaseTableImp.OnStop()
	log.Debug("Table", "OnStop")
	//将游戏状态设置到空档期
	self.fsm.Call(VoidPeriodEvent)
	self.NotifyStop()
	self.ExecuteCallBackMsg() //统一发送数据到客户端
	for _, player := range self.seats {
		player.OnUnBind()
	}

	var nv *list.Element
	for e := self.viewer.Front(); e != nil; e = nv {
		nv = e.Next()
		self.viewer.Remove(e)
	}
}
func (self *Table) OnDestroy() {
	self.BaseTableImp.OnDestroy()
	log.Debug("BaseTableImp", "OnDestroy")
	self.stoped = true
}

func (self *Table) onGameOver() {
	self.Finish()
}

/*
*
牌桌主循环
定帧计算所有玩家的位置
*/
func (self *Table) Update(arge interface{}) {
	self.ExecuteEvent(arge) //执行这一帧客户端发送过来的消息
	if self.State() == room.Active {
		self.current_frame++

		if self.current_frame-self.sync_frame >= 1 {
			//每帧同步一次
			self.sync_frame = self.current_frame
			self.StateSwitch()
		}

		ready := true
		for _, seat := range self.GetSeats() {
			if seat.Bind() == false {
				//还没有准备好
				ready = false
				break
			}
		}
		if ready == false {
			//有玩家离开了牌桌,牌桌退出
			self.Finish()
		}

	} else if self.State() == room.Initialized {
		ready := true
		for _, seat := range self.GetSeats() {
			if seat.SitDown() == false {
				//还没有准备好
				ready = false
				break
			}
		}
		if ready {
			self.Start() //开始游戏了
		}
	}

	self.ExecuteCallBackMsg() //统一发送数据到客户端
	self.CheckTimeOut()
	if !self.stoped {
		timewheel.GetTimeWheel().AddTimer(100*time.Millisecond, nil, self.Update)
	}
}

func (self *Table) Exit(session gate.Session) error {
	player := self.GetBindPlayer(session)
	if player != nil {
		playerImp := player.(*objects.Player)
		playerImp.OnUnBind()
		return nil
	}
	return nil
}

func (self *Table) getSeatsMap() []map[string]interface{} {
	m := make([]map[string]interface{}, len(self.seats))
	for i, player := range self.seats {
		if player != nil {
			m[i] = player.SerializableMap()
		}
	}
	return m
}

/*
*
玩家获取关卡信息
*/
func (self *Table) getLevel(session gate.Session) {
	playerImp := self.GetBindPlayer(session)
	if playerImp != nil {
		player := playerImp.(*objects.Player)
		player.OnRequest(session)
		player.OnSitDown()
	}
}
