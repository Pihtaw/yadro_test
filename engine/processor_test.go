package engine

import (
	"strings"
	"testing"
)

// TestExampleFromReadme verifies the sample scenario from README
func TestExampleFromReadme(t *testing.T) {
	cfg := Config{Floors: 2, Monsters: 2, OpenAt: "14:05:00", Duration: 2}
	events := `[14:00:00] 1 1
[14:00:00] 2 1
[14:10:00] 2 2
[14:10:00] 3 2
[14:11:00] 2 5
[14:12:00] 3 3
[14:14:00] 2 3
[14:27:00] 2 11 60
[14:29:00] 2 11 50
[14:40:00] 1 2
[14:41:00] 1 3
[14:44:00] 1 11 50
[14:45:00] 1 3
[14:48:00] 1 4
[14:48:00] 1 6
[14:49:00] 1 11 25
[14:49:02] 1 10 80
[14:50:00] 1 11 65
[14:59:00] 1 7
[15:04:00] 1 8
`
	out, err := Process(cfg, events)
	if err != nil {
		t.Fatalf("Process error: %v", err)
	}
	expect := []string{
		"[14:48:00] Player [1] entered the boss's floor",
		"[14:59:00] Player [1] killed the boss",
		"[SUCCESS] 1",
		"[FAIL] 2",
		"[DISQUAL] 3",
	}
	for _, e := range expect {
		if !strings.Contains(out, e) {
			t.Fatalf("expected %q in output, got:\n%s", e, out)
		}
	}
}

// TestDisqualBeforeOpen ensures entering before open disqualifies
func TestDisqualBeforeOpen(t *testing.T) {
	cfg := Config{Floors: 1, Monsters: 1, OpenAt: "10:00:00", Duration: 1}
	events := `[09:59:59] 1 1
[09:59:59] 1 2
`
	out, err := Process(cfg, events)
	if err != nil {
		t.Fatalf("Process error: %v", err)
	}
	if !strings.Contains(out, "is disqualified") {
		t.Fatalf("expected disqualification, got:\n%s", out)
	}
	if !strings.Contains(out, "[DISQUAL] 1") {
		t.Fatalf("expected final DISQUAL, got:\n%s", out)
	}
}

// TestDeathStopsProcessing ensures events after death are ignored
func TestDeathStopsProcessing(t *testing.T) {
	cfg := Config{Floors: 1, Monsters: 1, OpenAt: "10:00:00", Duration: 1}
	events := `[10:00:00] 1 1
[10:01:00] 1 2
[10:02:00] 1 11 200
[10:03:00] 1 3
`
	out, err := Process(cfg, events)
	if err != nil {
		t.Fatalf("Process error: %v", err)
	}
	if strings.Contains(out, "[10:03:00] Player [1] killed the monster") {
		t.Fatalf("event after death was processed:\n%s", out)
	}
	if !strings.Contains(out, "[FAIL] 1") {
		t.Fatalf("expected FAIL after death, got:\n%s", out)
	}
}

// TestDisqualIgnoreSubsequent ensures events after disqual are ignored
func TestDisqualIgnoreSubsequent(t *testing.T) {
	cfg := Config{Floors: 1, Monsters: 1, OpenAt: "10:00:00", Duration: 1}
	events := `[10:00:00] 1 1
[10:01:00] 1 9 reason
[10:02:00] 1 2
[10:03:00] 1 3
`
	out, err := Process(cfg, events)
	if err != nil {
		t.Fatalf("Process error: %v", err)
	}
	if strings.Contains(out, "entered the dungeon") {
		t.Fatalf("enter after disqual should be ignored:\n%s", out)
	}
	if !strings.Contains(out, "[DISQUAL] 1") {
		t.Fatalf("expected DISQUAL final state, got:\n%s", out)
	}
}

// TestFloorTimeAccounting ensures floor times recorded only when cleared
func TestFloorTimeAccounting(t *testing.T) {
	cfg := Config{Floors: 2, Monsters: 1, OpenAt: "10:00:00", Duration: 1}
	events := `[10:00:00] 1 1
[10:01:00] 1 2
[10:02:00] 1 3
[10:03:00] 1 4
[10:04:00] 1 3
[10:05:00] 1 6
[10:06:00] 1 7
[10:07:00] 1 8
`
	out, err := Process(cfg, events)
	if err != nil {
		t.Fatalf("Process error: %v", err)
	}
	// first floor cleared at 10:02 -> time 00:01:00
	if !strings.Contains(out, "00:01:00") {
		t.Fatalf("expected first floor time 00:01:00 in output, got:\n%s", out)
	}
	// boss time should be 00:01:00 (entered 10:05, killed 10:06)
	if !strings.Contains(out, "00:01:00") {
		t.Fatalf("expected boss time 00:01:00 in output, got:\n%s", out)
	}
}
