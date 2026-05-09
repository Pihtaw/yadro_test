package engine

import (
	"bufio"
	"strconv"
	"strings"
)

// ParseEvents parses raw events text into Event slices.
// Lines that cannot be parsed are ignored.
func ParseEvents(raw string) []Event {
	events := []Event{}
	scanner := bufio.NewScanner(strings.NewReader(raw))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "]", 2)
		if len(parts) != 2 {
			continue
		}
		tsStr := strings.TrimPrefix(parts[0], "[")
		rest := strings.TrimSpace(parts[1])
		fields := strings.Fields(rest)
		if len(fields) < 2 {
			continue
		}
		id, err1 := strconv.Atoi(fields[0])
		eid, err2 := strconv.Atoi(fields[1])
		if err1 != nil || err2 != nil {
			continue
		}
		param := ""
		if len(fields) > 2 {
			param = strings.Join(fields[2:], " ")
		}
		events = append(events, Event{
			Time:  tsStr,
			ID:    eid,
			Param: param,
			Raw:   line,
		})
		// store player id in Raw for later parsing convenience (we'll reparse when applying)
		_ = id // player id will be parsed again in processor.ApplyEvent
	}
	return events
}
