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
    "github.com/shangzongyu/mqant/utils"
    "github.com/shangzongyu/mqantserver/server/xaxb/objects"
    "math"
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

func (self *Table) InitFsm() {
    self.fsm = *NewFSM(VoidPeriod)
    self.VoidPeriodHandler = FSMHandler(func() FSMState {
        fmt.Println("已进入空档期")
        return VoidPeriod
    })
    self.IdlePeriodHandler = FSMHandler(func() FSMState {
        fmt.Println("已进入空闲期")
        self.step1 = self.current_frame
        self.NotifyIdle()

        for _, seat := range self.GetSeats() {
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
    self.BettingPeriodHandler = FSMHandler(func() FSMState {
        fmt.Println("已进入押注期")
        self.step2 = self.current_frame
        self.NotifyBetting()
        return BettingPeriod
    })
    self.OpeningPeriodHandler = FSMHandler(func() FSMState {
        fmt.Println("已进入开奖期")
        self.step3 = self.current_frame
        self.NotifyOpening()
        return OpeningPeriod
    })
    self.SettlementPeriodHandler = FSMHandler(func() FSMState {
        fmt.Println("已进入结算期")
        var mixWeight int64 = math.MaxInt64
        var winer *objects.Player = nil
        Result := utils.RandInt64(0, 10)
        for _, seat := range self.GetSeats() {
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

        self.step4 = self.current_frame
        self.NotifySettlement(Result)
        return SettlementPeriod
    })

    self.fsm.AddHandler(IdlePeriod, VoidPeriodEvent, self.VoidPeriodHandler)
    self.fsm.AddHandler(SettlementPeriod, VoidPeriodEvent, self.VoidPeriodHandler)
    self.fsm.AddHandler(BettingPeriod, VoidPeriodEvent, self.VoidPeriodHandler)
    self.fsm.AddHandler(OpeningPeriod, VoidPeriodEvent, self.VoidPeriodHandler)

    self.fsm.AddHandler(VoidPeriod, IdlePeriodEvent, self.IdlePeriodHandler)
    self.fsm.AddHandler(SettlementPeriod, IdlePeriodEvent, self.IdlePeriodHandler)

    self.fsm.AddHandler(IdlePeriod, BettingPeriodEvent, self.BettingPeriodHandler)
    self.fsm.AddHandler(BettingPeriod, OpeningPeriodEvent, self.OpeningPeriodHandler)
    self.fsm.AddHandler(OpeningPeriod, SettlementPeriodEvent, self.SettlementPeriodHandler)
}

/*
*
进入空闲期
*/
func (self *Table) StateSwitch() {
    switch self.fsm.getState() {
    case VoidPeriod:

    case IdlePeriod:
        if (self.current_frame - self.step1) > 5 {
            self.fsm.Call(BettingPeriodEvent)
        } else {
            //this.NotifyAxes()
        }
    case BettingPeriod:
        if (self.current_frame - self.step2) > 20 {
            self.fsm.Call(OpeningPeriodEvent)
        } else {
            ready := true
            for _, seat := range self.GetSeats() {
                player := seat.(*objects.Player)
                if player.SitDown() && !player.Stake {
                    ready = false
                }
            }
            if ready {
                //都押注了直接开奖
                self.fsm.Call(OpeningPeriodEvent)
            }
        }
    case OpeningPeriod:
        if (self.current_frame - self.step3) > 5 {
            self.fsm.Call(SettlementPeriodEvent)
        } else {
            //this.NotifyAxes()
        }
    case SettlementPeriod:
        if (self.current_frame - self.step4) > 5 {
            self.fsm.Call(IdlePeriodEvent)
        } else {
            //this.NotifyAxes()
        }
    }
}
