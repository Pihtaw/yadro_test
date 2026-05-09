package engine

import "time"

type Config struct {
	Floors   int    `json:"Floors"`
	Monsters int    `json:"Monsters"`
	OpenAt   string `json:"OpenAt"`
	Duration int    `json:"Duration"` // hours
}

type Event struct {
	Time  string
	ID    int
	Param string
	Raw   string
}

type Player struct {
	ID                int
	Registered        bool
	Inside            bool
	EnterAt           *time.Time
	LeaveAt           *time.Time
	HP                int
	Dead              bool
	Disqualified      bool
	DisqualReason     string
	CurrentFloor      int
	MonstersRemaining []int
	FloorStart        *time.Time
	AccumFloorTimes   []time.Duration
	EnteredBossAt     *time.Time
	BossKilledAt      *time.Time
}

type ReportRow struct {
	State string
	ID    int
	Line  string
}
