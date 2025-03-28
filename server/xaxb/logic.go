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

package xaxb

import (
	"fmt"
	"math"

	"github.com/shangzongyu/mqant/utils"
	"github.com/shangzongyu/mqantserver/server/xaxb/objects"
)

var (
	VoidPeriod            = FSMState("空档期")
	IdlePeriod            = FSMState("空闲期")
	BettingPeriod         = FSMState("押注期")
	OpeningPeriod         = FSMState("开奖期")
	SettlementPeriod      = FSMState("结算期")
	VoidPeriodEvent       = FSMEvent("进入空档期")
	IdlePeriodEvent       = FSMEvent("进入空闲期")
	BettingPeriodEvent    = FSMEvent("进入押注期")
	OpeningPeriodEvent    = FSMEvent("进入开奖期")
	SettlementPeriodEvent = FSMEvent("进入结算期")
)

func (t *Table) InitFsm() {
	t.fsm = *NewFSM(VoidPeriod)
	t.VoidPeriodHandler = FSMHandler(func() FSMState {
		fmt.Println("已进入空档期")
		return VoidPeriod
	})
	t.IdlePeriodHandler = FSMHandler(func() FSMState {
		fmt.Println("已进入空闲期")
		t.step1 = t.currentFrame
		t.NotifyIdle()

		for _, seat := range t.GetSeats() {
			player := seat.(*objects.Player)
			if player.Bind() {
				if player.Coin <= 0 {
					player.Session().Send("XaXb/Exit", []byte(`{"Info":"金币不足你被强制离开房间"}`))
					player.OnUnBind() //踢下线
				}
			}
		}

		return IdlePeriod
	})
	t.BettingPeriodHandler = FSMHandler(func() FSMState {
		fmt.Println("已进入押注期")
		t.step2 = t.currentFrame
		t.NotifyBetting()
		return BettingPeriod
	})
	t.OpeningPeriodHandler = FSMHandler(func() FSMState {
		fmt.Println("已进入开奖期")
		t.step3 = t.currentFrame
		t.NotifyOpening()
		return OpeningPeriod
	})
	t.SettlementPeriodHandler = FSMHandler(func() FSMState {
		fmt.Println("已进入结算期")
		var mixWeight int64 = math.MaxInt64
		var winer *objects.Player = nil
		Result := utils.RandInt64(0, 10)
		for _, seat := range t.GetSeats() {
			player := seat.(*objects.Player)
			if player.Stake {
				player.Weight = int64(math.Abs(float64(player.Target - Result)))
				if mixWeight > player.Weight {
					mixWeight = player.Weight
					winer = player
				}
			}
		}
		if winer != nil {
			winer.Coin += 800
		}

		t.step4 = t.currentFrame
		t.NotifySettlement(Result)
		return SettlementPeriod
	})

	t.fsm.AddHandler(IdlePeriod, VoidPeriodEvent, t.VoidPeriodHandler)
	t.fsm.AddHandler(SettlementPeriod, VoidPeriodEvent, t.VoidPeriodHandler)
	t.fsm.AddHandler(BettingPeriod, VoidPeriodEvent, t.VoidPeriodHandler)
	t.fsm.AddHandler(OpeningPeriod, VoidPeriodEvent, t.VoidPeriodHandler)

	t.fsm.AddHandler(VoidPeriod, IdlePeriodEvent, t.IdlePeriodHandler)
	t.fsm.AddHandler(SettlementPeriod, IdlePeriodEvent, t.IdlePeriodHandler)

	t.fsm.AddHandler(IdlePeriod, BettingPeriodEvent, t.BettingPeriodHandler)
	t.fsm.AddHandler(BettingPeriod, OpeningPeriodEvent, t.OpeningPeriodHandler)
	t.fsm.AddHandler(OpeningPeriod, SettlementPeriodEvent, t.SettlementPeriodHandler)
}

/*
*
进入空闲期
*/
func (t *Table) StateSwitch() {
	switch t.fsm.getState() {
	case VoidPeriod:

	case IdlePeriod:
		if (t.currentFrame - t.step1) > 5 {
			t.fsm.Call(BettingPeriodEvent)
		} else {
			//this.NotifyAxes()
		}
	case BettingPeriod:
		if (t.currentFrame - t.step2) > 20 {
			t.fsm.Call(OpeningPeriodEvent)
		} else {
			ready := true
			for _, seat := range t.GetSeats() {
				player := seat.(*objects.Player)
				if player.SitDown() && !player.Stake {
					ready = false
				}
			}
			if ready {
				//都押注了直接开奖
				t.fsm.Call(OpeningPeriodEvent)
			}
		}
	case OpeningPeriod:
		if (t.currentFrame - t.step3) > 5 {
			t.fsm.Call(SettlementPeriodEvent)
		} else {
			//this.NotifyAxes()
		}
	case SettlementPeriod:
		if (t.currentFrame - t.step4) > 5 {
			t.fsm.Call(IdlePeriodEvent)
		} else {
			//this.NotifyAxes()
		}
	}
}
