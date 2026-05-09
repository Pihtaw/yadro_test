package engine

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

// LoadConfig reads config.json from path and returns Config.
func LoadConfig(path string) (Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return Config{}, err
	}
	defer f.Close()
	var cfg Config
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Process is the high-level entry point: given config and raw events text,
// returns the full output (log + final report).
// Uses forgiving semantics: event 6 forces boss entry.
func Process(cfg Config, rawEvents string) (string, error) {
	return ProcessWithTimeProvider(cfg, rawEvents, RealTimeProvider{})
}

// ProcessWithTimeProvider same as Process but allows injecting a TimeProvider for tests.
func ProcessWithTimeProvider(cfg Config, rawEvents string, tp TimeProvider) (string, error) {
	if cfg.Floors < 1 {
		return "", fmt.Errorf("Floors must be >= 1")
	}
	if cfg.Monsters < 0 {
		return "", fmt.Errorf("Monsters must be >= 0")
	}
	openAt, err := tp.Parse("15:04:05", cfg.OpenAt)
	if err != nil {
		return "", fmt.Errorf("invalid OpenAt: %w", err)
	}
	closeAt := openAt.Add(time.Duration(cfg.Duration) * time.Hour)

	players := map[int]*Player{}
	outLines := []string{}

	scanner := bufio.NewScanner(strings.NewReader(rawEvents))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		tsStr, playerID, eid, param, ok := parseLine(line)
		if !ok {
			continue
		}

		t, err := tp.Parse("15:04:05", tsStr)
		if err != nil {
			continue
		}

		p := ensurePlayer(players, playerID, cfg)

		if p.Disqualified && eid != 1 {
			continue
		}
		if p.Dead {
			continue
		}

		applyEvent(&outLines, p, cfg, playerID, eid, param, t, tsStr, openAt, closeAt)
	}
	buildFinalReport(&outLines, players, cfg, closeAt)
	return strings.Join(outLines, "\n") + "\n", nil
}

// parseLine parses a single raw line like "[HH:MM:SS] id event [param...]".
// Returns timestamp string, player id (as int), event id, param string and ok flag.
func parseLine(line string) (string, int, int, string, bool) {
	parts := strings.SplitN(line, "]", 2)
	if len(parts) != 2 {
		return "", 0, 0, "", false
	}
	tsStr := strings.TrimPrefix(parts[0], "[")
	rest := strings.TrimSpace(parts[1])
	fields := strings.Fields(rest)
	if len(fields) < 2 {
		return "", 0, 0, "", false
	}
	playerID, err1 := strconv.Atoi(fields[0])
	eid, err2 := strconv.Atoi(fields[1])
	if err1 != nil || err2 != nil {
		return "", 0, 0, "", false
	}
	param := ""
	if len(fields) > 2 {
		param = strings.Join(fields[2:], " ")
	}
	return tsStr, playerID, eid, param, true
}

// ensurePlayer returns existing player or creates and initializes a new one.
func ensurePlayer(players map[int]*Player, id int, cfg Config) *Player {
	if p, ok := players[id]; ok {
		return p
	}
	p := &Player{
		ID:                id,
		HP:                100,
		MonstersRemaining: make([]int, cfg.Floors),
		AccumFloorTimes:   make([]time.Duration, cfg.Floors),
	}
	for i := 0; i < cfg.Floors; i++ {
		p.MonstersRemaining[i] = cfg.Monsters
	}
	players[id] = p
	return p
}

// applyEvent applies a single event to player state and appends any log lines to outLines.
func applyEvent(outLines *[]string, p *Player, cfg Config, playerID int, eid int, param string, t time.Time, tsStr string, openAt, closeAt time.Time) {
	switch eid {
	case 1: // register
		p.Registered = true
		*outLines = append(*outLines, fmt.Sprintf("[%s] Player [%d] registered", tsStr, playerID))
	case 2: // enter dungeon
		applyEnter(outLines, p, cfg, playerID, t, tsStr, openAt, closeAt)
	case 3: // killed monster
		applyKillMonster(outLines, p, cfg, playerID, t, tsStr)
	case 4: // next floor
		applyNextFloor(outLines, p, cfg, playerID, t, tsStr)
	case 5: // previous floor
		applyPrevFloor(outLines, p, cfg, playerID, t, tsStr)
	case 6: // entered boss floor (forgiving)
		applyEnterBoss(outLines, p, cfg, playerID, t, tsStr)
	case 7: // killed boss
		applyKillBoss(outLines, p, cfg, playerID, t, tsStr)
	case 8: // left dungeon
		applyLeave(outLines, p, cfg, playerID, t, tsStr)
	case 9: // cannot continue -> disqualify
		*outLines = append(*outLines, fmt.Sprintf("[%s] Player [%d] is disqualified", tsStr, playerID))
		p.Disqualified = true
		p.DisqualReason = param
	case 10: // heal
		applyHeal(outLines, p, playerID, param, tsStr)
	case 11: // damage
		applyDamage(outLines, p, playerID, param, t, tsStr)
	default:
	}
}

func applyEnter(outLines *[]string, p *Player, cfg Config, playerID int, t time.Time, tsStr string, openAt, closeAt time.Time) {
	if !p.Registered {
		*outLines = append(*outLines, fmt.Sprintf("[%s] Player [%d] is disqualified", tsStr, playerID))
		p.Disqualified = true
		p.DisqualReason = "not registered"
		return
	}
	if p.Inside {
		*outLines = append(*outLines, fmt.Sprintf("[%s] Player [%d] makes imposible move [2]", tsStr, playerID))
		return
	}
	if t.Before(openAt) {
		*outLines = append(*outLines, fmt.Sprintf("[%s] Player [%d] is disqualified", tsStr, playerID))
		p.Disqualified = true
		p.DisqualReason = "entered before opening"
		return
	}
	if !t.Before(closeAt) {
		*outLines = append(*outLines, fmt.Sprintf("[%s] Player [%d] is disqualified", tsStr, playerID))
		p.Disqualified = true
		p.DisqualReason = "entered after closing"
		return
	}
	p.Inside = true
	tt := t
	p.EnterAt = &tt
	p.CurrentFloor = 1
	fs := t
	p.FloorStart = &fs
	*outLines = append(*outLines, fmt.Sprintf("[%s] Player [%d] entered the dungeon", tsStr, playerID))
}

func applyKillMonster(outLines *[]string, p *Player, cfg Config, playerID int, t time.Time, tsStr string) {
	if !p.Inside || p.CurrentFloor < 1 || p.CurrentFloor > cfg.Floors {
		*outLines = append(*outLines, fmt.Sprintf("[%s] Player [%d] makes imposible move [3]", tsStr, playerID))
		return
	}
	idx := p.CurrentFloor - 1
	if p.MonstersRemaining[idx] <= 0 {
		*outLines = append(*outLines, fmt.Sprintf("[%s] Player [%d] makes imposible move [3]", tsStr, playerID))
		return
	}
	p.MonstersRemaining[idx]--
	*outLines = append(*outLines, fmt.Sprintf("[%s] Player [%d] killed the monster", tsStr, playerID))
	if p.MonstersRemaining[idx] == 0 {
		if p.FloorStart != nil {
			p.AccumFloorTimes[idx] = t.Sub(*p.FloorStart)
			p.FloorStart = nil
		}
	}
}

func applyNextFloor(outLines *[]string, p *Player, cfg Config, playerID int, t time.Time, tsStr string) {
	if !p.Inside {
		*outLines = append(*outLines, fmt.Sprintf("[%s] Player [%d] makes imposible move [4]", tsStr, playerID))
		return
	}
	if p.CurrentFloor < 1 {
		*outLines = append(*outLines, fmt.Sprintf("[%s] Player [%d] makes imposible move [4]", tsStr, playerID))
		return
	}
	if p.CurrentFloor <= cfg.Floors {
		idx := p.CurrentFloor - 1
		if p.MonstersRemaining[idx] > 0 {
			*outLines = append(*outLines, fmt.Sprintf("[%s] Player [%d] makes imposible move [4]", tsStr, playerID))
			return
		}
	}
	p.CurrentFloor++
	if p.CurrentFloor > cfg.Floors {
		p.CurrentFloor = cfg.Floors + 1
		tt := t
		if p.EnteredBossAt == nil {
			p.EnteredBossAt = &tt
			*outLines = append(*outLines, fmt.Sprintf("[%s] Player [%d] went to the next floor", tsStr, playerID))
			*outLines = append(*outLines, fmt.Sprintf("[%s] Player [%d] entered the boss's floor", tsStr, playerID))
		} else {
			*outLines = append(*outLines, fmt.Sprintf("[%s] Player [%d] went to the next floor", tsStr, playerID))
		}
		p.FloorStart = nil
	} else {
		if p.MonstersRemaining[p.CurrentFloor-1] > 0 {
			fs := t
			p.FloorStart = &fs
		} else {
			p.FloorStart = nil
		}
		*outLines = append(*outLines, fmt.Sprintf("[%s] Player [%d] went to the next floor", tsStr, playerID))
	}
}

func applyPrevFloor(outLines *[]string, p *Player, cfg Config, playerID int, t time.Time, tsStr string) {
	if !p.Inside {
		*outLines = append(*outLines, fmt.Sprintf("[%s] Player [%d] makes imposible move [5]", tsStr, playerID))
		return
	}
	if p.CurrentFloor <= 1 {
		*outLines = append(*outLines, fmt.Sprintf("[%s] Player [%d] makes imposible move [5]", tsStr, playerID))
		return
	}
	p.CurrentFloor--
	if p.CurrentFloor <= cfg.Floors && p.MonstersRemaining[p.CurrentFloor-1] > 0 {
		fs := t
		p.FloorStart = &fs
	} else {
		p.FloorStart = nil
	}
	*outLines = append(*outLines, fmt.Sprintf("[%s] Player [%d] went to the previous floor", tsStr, playerID))
}

func applyEnterBoss(outLines *[]string, p *Player, cfg Config, playerID int, t time.Time, tsStr string) {
	if !p.Inside {
		*outLines = append(*outLines, fmt.Sprintf("[%s] Player [%d] makes imposible move [6]", tsStr, playerID))
		return
	}
	if p.CurrentFloor != cfg.Floors+1 {
		p.CurrentFloor = cfg.Floors + 1
	}
	if p.EnteredBossAt == nil {
		tt := t
		p.EnteredBossAt = &tt
	}
	p.FloorStart = nil
	*outLines = append(*outLines, fmt.Sprintf("[%s] Player [%d] entered the boss's floor", tsStr, playerID))
}

func applyKillBoss(outLines *[]string, p *Player, cfg Config, playerID int, t time.Time, tsStr string) {
	if p.EnteredBossAt == nil || p.CurrentFloor != cfg.Floors+1 {
		*outLines = append(*outLines, fmt.Sprintf("[%s] Player [%d] makes imposible move [7]", tsStr, playerID))
		return
	}
	tt := t
	p.BossKilledAt = &tt
	*outLines = append(*outLines, fmt.Sprintf("[%s] Player [%d] killed the boss", tsStr, playerID))
}

func applyLeave(outLines *[]string, p *Player, cfg Config, playerID int, t time.Time, tsStr string) {
	if !p.Inside {
		*outLines = append(*outLines, fmt.Sprintf("[%s] Player [%d] makes imposible move [8]", tsStr, playerID))
		return
	}
	tt := t
	p.LeaveAt = &tt
	p.Inside = false
	p.FloorStart = nil
	*outLines = append(*outLines, fmt.Sprintf("[%s] Player [%d] left the dungeon", tsStr, playerID))
}

func applyHeal(outLines *[]string, p *Player, playerID int, param string, tsStr string) {
	val, _ := strconv.Atoi(param)
	if val < 0 {
		val = 0
	}
	p.HP += val
	if p.HP > 100 {
		p.HP = 100
	}
	*outLines = append(*outLines, fmt.Sprintf("[%s] Player [%d] has restored [%d] of health", tsStr, playerID, val))
}

func applyDamage(outLines *[]string, p *Player, playerID int, param string, t time.Time, tsStr string) {
	val, _ := strconv.Atoi(param)
	if val < 0 {
		val = 0
	}
	p.HP -= val
	*outLines = append(*outLines, fmt.Sprintf("[%s] Player [%d] recieved [%d] of damage", tsStr, playerID, val))
	if p.HP <= 0 {
		p.HP = 0
		p.Dead = true
		if p.Inside && p.LeaveAt == nil {
			tt := t
			p.LeaveAt = &tt
			p.Inside = false
		}
		*outLines = append(*outLines, fmt.Sprintf("[%s] Player [%d] is dead", tsStr, playerID))
	}
}

// buildFinalReport composes the final report lines and appends them to outLines.
func buildFinalReport(outLines *[]string, players map[int]*Player, cfg Config, closeAt time.Time) {
	*outLines = append(*outLines, "")
	*outLines = append(*outLines, "Final report:")

	type row struct {
		state string
		id    int
		line  string
		order int
	}
	rows := []row{}

	for id, p := range players {
		state := "FAIL"
		if p.Disqualified {
			state = "DISQUAL"
		} else if p.BossKilledAt != nil {
			state = "SUCCESS"
		} else {
			state = "FAIL"
		}

		var timeInDungeon time.Duration
		if p.EnterAt == nil {
			timeInDungeon = 0
		} else {
			end := closeAt
			if p.LeaveAt != nil && p.LeaveAt.Before(end) {
				end = *p.LeaveAt
			}
			if end.After(*p.EnterAt) {
				timeInDungeon = end.Sub(*p.EnterAt)
			} else {
				timeInDungeon = 0
			}
		}

		arr := []string{fmtDur(timeInDungeon)}
		for i := 0; i < cfg.Floors-1; i++ {
			if p.AccumFloorTimes[i] > 0 {
				arr = append(arr, fmtDur(p.AccumFloorTimes[i]))
			} else {
				arr = append(arr, "00:00:00")
			}
		}
		bossTime := time.Duration(0)
		if p.EnteredBossAt != nil && p.BossKilledAt != nil {
			bossTime = p.BossKilledAt.Sub(*p.EnteredBossAt)
			if bossTime < 0 {
				bossTime = 0
			}
		}
		arr = append(arr, fmtDur(bossTime))

		line := fmt.Sprintf("[%s] %d [%s] HP:%d", state, id, strings.Join(arr, ", "), p.HP)
		order := 2
		if state == "SUCCESS" {
			order = 0
		} else if state == "DISQUAL" {
			order = 3
		}
		rows = append(rows, row{state: state, id: id, line: line, order: order})
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].order != rows[j].order {
			return rows[i].order < rows[j].order
		}
		return rows[i].id < rows[j].id
	})
	for _, r := range rows {
		*outLines = append(*outLines, r.line)
	}
}

// fmtDur formats duration as HH:MM:SS with leading zeros.
func fmtDur(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}
