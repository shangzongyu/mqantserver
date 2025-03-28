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

package objects

import (
	"github.com/shangzongyu/mqant/gate"
	"math"
	"time"
)

type Player struct {
	Session         gate.Session
	Rid             string //角色ID  userid
	SitDown         bool   //是否已坐下 ,如果网络断开会设置为false,当网络连接成功以后需要重新坐下
	NetBroken       int64  //网络中断时间，超过60秒就踢出房间或者做其他处理 单位/秒
	LastRequestDate int64  //玩家最后一次请求时间	单位纳秒
	X               float64
	Y               float64
	Wid             int
	XSpeed          float64
	YSpeed          float64
	RotateDirection int // rotate direction: 1-clockwise, 2-counterclockwise
	BallRadius      float64
	Angle           float64
	Power           float64

	RotateSpeed int
	DegToRad    float64
	MinPower    float64
	MaxPower    float64
}

// OnRequest 玩家主动发请求时间
func (p *Player) OnRequest(session gate.Session) {
	p.Session = session
	p.LastRequestDate = time.Now().UnixNano()
}

func (p *Player) OnSitDown() {
	p.SitDown = true
}

func (p *Player) OnSitUp() {
	p.SitDown = false
}

func (p *Player) OnNetBroken() {
	p.NetBroken = time.Now().Unix()
}

func (p *Player) Move(friction float64) {
	p.X = p.X + p.XSpeed
	p.Y = p.Y + p.YSpeed
	// reduce ball speed using friction 速度递减
	p.XSpeed *= friction
	p.YSpeed *= friction
}

func (p *Player) Rotate() {
	p.Angle += float64(p.RotateSpeed * p.RotateDirection)
}

func (p *Player) Fire(X float64, Y float64, angle float64, power float64) {
	//发射
	p.XSpeed += math.Cos(angle*p.DegToRad) * power / 20
	p.YSpeed += math.Sin(angle*p.DegToRad) * power / 20
	p.Power = p.MinPower
	//p.Angle=angle  //这里不同步客户端发过来的角速度
	p.X = X
	p.Y = Y
	p.Power = power
	p.RotateDirection *= -1
}
