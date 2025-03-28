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
	"encoding/json"

	"github.com/shangzongyu/mqantserver/server/xaxb/objects"
)

// NotifyAxes 定期刷新所有玩家的位置
func (t *Table) NotifyAxes() {
	seats := make([]map[string]interface{}, 0)
	for _, player := range t.seats {
		if player.IsBind() {
			seats = append(seats, player.SerializableMap())
		}
	}

	b, _ := json.Marshal(map[string]interface{}{
		"State":     t.State(),
		"StateGame": t.fsm.getState(),
		"Seats":     seats,
	})

	t.NotifyCallBackMsg("XaXb/OnSync", b)
}

// NotifyJoin 通知所有玩家有新玩家加入

func (t *Table) NotifyJoin(player *objects.Player) {
	b, _ := json.Marshal(player.SerializableMap())
	t.NotifyCallBackMsg("XaXb/OnEnter", b)
}

// NotifyResume 通知所有玩家开始游戏了
func (t *Table) NotifyResume() {
	b, _ := json.Marshal(t.getSeatsMap())
	t.NotifyCallBackMsg("XaXb/OnResume", b)
}

// NotifyPause 通知所有玩家开始游戏了
func (t *Table) NotifyPause() {
	b, _ := json.Marshal(t.getSeatsMap())
	t.NotifyCallBackMsg("XaXb/OnPause", b)
}

// NotifyStop 通知所有玩家开始游戏了
func (t *Table) NotifyStop() {
	b, _ := json.Marshal(t.getSeatsMap())
	t.NotifyCallBackMsg("XaXb/OnStop", b)
}

// NotifyIdle 通知所有玩家进入空闲期了
func (t *Table) NotifyIdle() {
	b, _ := json.Marshal(map[string]interface{}{
		"Coin": 500,
	})
	t.NotifyCallBackMsg("XaXb/Idle", b)
}

// NotifyBetting 通知所有玩家开始押注了
func (t *Table) NotifyBetting() {
	b, _ := json.Marshal(map[string]interface{}{
		"Coin": 500,
	})

	t.NotifyCallBackMsg("XaXb/Betting", b)
}

// NotifyOpening 通知所有玩家开始开奖了
func (t *Table) NotifyOpening() {
	b, _ := json.Marshal(map[string]interface{}{
		"Coin": 500,
	})
	t.NotifyCallBackMsg("XaXb/Opening", b)
}

// NotifySettlement 通知所有玩家开奖结果出来了
func (t *Table) NotifySettlement(Result int64) {
	seats := make([]map[string]interface{}, 0)
	for _, player := range t.seats {
		if player.IsBind() {
			seats = append(seats, player.SerializableMap())
		}
	}
	b, _ := json.Marshal(map[string]interface{}{
		"Result": Result,
		"Seats":  seats,
	})
	t.NotifyCallBackMsg("XaXb/Settlement", b)
}
