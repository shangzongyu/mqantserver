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
	"sync"

	"github.com/shangzongyu/mqant/module"
)

type Room struct {
	module module.Module
	lock   *sync.RWMutex
	tables map[int]*Table
	index  int
	max    int
}

func NewRoom(module module.Module) *Room {
	room := &Room{
		module: module,
		lock:   new(sync.RWMutex),
		tables: map[int]*Table{},
		index:  0,
		max:    0,
	}
	return room
}

func (ro *Room) create(module module.Module) *Table {
	ro.lock.Lock()
	ro.index++
	table := NewTable(module, ro.index)
	ro.tables[ro.index] = table
	ro.lock.Unlock()
	return table
}

func (ro *Room) GetTable(tableId int) *Table {
	if table, ok := ro.tables[tableId]; ok {
		return table
	}
	return nil
}

func (ro *Room) GetEmptyTable() (*Table, error) {
	for _, table := range ro.tables {
		if table.Empty() {
			return table, nil
		}
	}
	//没有找到已创建的空房间,新创建一个
	table := ro.create(ro.module)
	if table == nil {
		return nil, fmt.Errorf("fail create table")
	}
	return table, nil
}
