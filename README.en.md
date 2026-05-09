# Project Overview

This repository contains a Go implementation of a dungeon event processor. The program reads timestamped events, updates player states, emits logs and produces a final report.

---

# Project Layout

- **main.go** — CLI entrypoint.
- **engine** — package with core logic: parser, event handlers, final report generation and tests.
- **data** — example input files: `config.json` and `events`.
- **Makefile** — convenient build/run/test targets.
- **go.mod** — Go module file.

---

# Algorithm Summary

**Short**  
The processor reads timestamped event lines, updates player states, emits log lines and produces a final report.

**Core types**
- **Config** — parameters: number of floors, monsters per floor, open time and duration.
- **Player** — player state: registered flag, inside flag, HP, current floor, enter/leave times, per-floor clear times, boss times, dead/disqualified flags.
- **Event** — a single input line: timestamp, player id, event code and optional parameter.

**Supported events**
- `1` — register
- `2` — enter dungeon
- `3` — kill monster
- `4` — next floor
- `5` — previous floor
- `6` — enter boss floor (forgiving mode forces boss entry)
- `7` — kill boss
- `8` — leave dungeon
- `9` — cannot continue → disqualify
- `10` — heal (parameter is amount)
- `11` — damage (parameter is amount)

**Key rules**
- Events before registration cause disqualification.
- Entering before open or after close causes disqualification.
- After death or disqualification subsequent events (except registration) are ignored.
- Forgiving mode: event `6` forces the player to the boss floor.
- Floor clear time is recorded when the last monster on that floor is killed.
- Final report lists player status (`SUCCESS`, `FAIL`, `DISQUAL`), total time in dungeon, per-floor times and HP.

---

# Input Formats

**config.json** example:
```json
{
  "Floors": 2,
  "Monsters": 2,
  "OpenAt": "14:05:00",
  "Duration": 2
}
```

**events** example line:
```
[14:10:00] 2 2
```
Meaning:
- `[HH:MM:SS]` — timestamp
- `2` — player id (integer)
- `2` — event code
- optional parameters follow

---

# Build and Run

**Build**
```bash
go build -o dungeon .
```

**Run with example data**
```bash
cp -f data/config.json config.json
./dungeon < data/events
rm -f config.json
```

**Makefile targets**
```bash
make build   # build binary
make run     # run ./dungeon < data/events
make test    # run all tests
make clean   # remove binary
```

---

# Testing

```bash
go test ./...
```

Tests include integration scenario and unit tests for:
- death handling and ignoring subsequent events;
- disqualification rules;
- floor time accounting and boss timing.

---
