# Logging Guidelines

> CLI logs, Web runtime events, and user-visible notification conventions.

---

## Overview

Traffic Consumer does not use a structured logging library. The maintained Go
backend uses callback-based logging so the same consumer can run in CLI mode and
Web mode:

- CLI passes a logger that prints plain messages to stdout.
- Web runtime stores logs in memory, mirrors them to the backend console, and
  exposes filtered increments through `/api/activity`.
- Web notifications are stored as events and exposed through `/api/activity`.

Keep this split. `internal/consumer` should not import HTTP or frontend code.

---

## Log Levels And Events

Use the level strings already present in `model.LogEntry` and `model.Event`:

- `info`: normal lifecycle messages such as start, stop, scheduler status, and
  history updates.
- `warning`: recoverable conditions that deserve user attention.
- `error`: failed downloads, invalid URLs, storage write failures, and runtime
  conflicts that block an operation.
- `success`: only for UI notifications that explicitly represent a completed
  user action.

Examples from the current Go path:

```go
c.log("info", "流量消耗器启动", "#0dcaf0")
c.log("error", "保存统计数据失败: "+err.Error(), "#dc3545")
```

Web event payloads use the same business messages that the UI displays:

```go
m.AddEvent(model.Event{
    Type: "status_update",
    Title: "状态更新：",
    Variant: stateVariant(state),
    Message: message,
})
```

---

## Runtime Buffers

`runtime.Manager` owns in-memory log and event buffers:

- `AddLog` assigns monotonically increasing `Seq` values, normalizes log source
  and level, and prints the same entry to the backend console.
- `AddEvent` assigns monotonically increasing `Seq` values and normalizes
  missing event types to `notification`.
- Both buffers keep only the latest 500 records.
- `LogEntry.Source` is the authoritative source dimension for frontend filters:
  `scheduler`, `runtime`, `download`.
- HTTP clients poll `/api/activity` with `after_log` / `after_event`; log
  reads may additionally send `include_logs`, `log_sources`, and `log_levels`.
- When `include_logs=false`, the handler must preserve the client's log cursor
  so pausing the log panel does not silently consume backlog.

This replaces deprecated Python Socket.IO pushes. Do not add new Socket.IO
behavior to the Python backend.

---

## ANSI And Color Handling

Go Web logs strip ANSI control sequences before storing messages:

```go
Message: stripANSI(message)
```

Colors are currently stored as simple CSS values for frontend compatibility.
Avoid adding terminal color codes to messages that will be shown in the Web UI.
If a CLI-only color feature is added later, strip or translate it before it
enters `model.LogEntry`.

---

## What To Log

Log or emit events for:

- Consumer start and stop.
- Scheduler registration, start, trigger, completion, next-run refresh, stop,
  and plan replacement.
- Retry exhaustion and failed URL removal.
- Stats write failures.
- User-triggered runtime operations that change state.
- Auto-start results when Web mode launches saved configs.

Do not log every chunk read from a response body. The consumer reads in 256 KB
chunks, and per-chunk logs would make the Web buffer useless.

---

## What Not To Log

Avoid logging secrets and sensitive request data:

- Do not log full custom request headers. Headers can carry auth tokens,
  cookies, or private referers.
- Do not log `headers-json` raw input on parse failures.
- Do not log local filesystem paths beyond config/stat file errors needed to
  diagnose storage failures.
- Do not include raw stack traces in user-visible HTTP responses.

URLs are currently user-visible because the application is built around URL
management, but be careful not to duplicate very long URLs across multiple log
streams.

---

## Common Mistakes

- Adding direct `fmt.Println` calls inside `internal/consumer` instead of using
  its `Logger` callback.
- Letting the frontend “pause logs” by continuing to advance the log cursor
  while dropping unseen log entries.
- Emitting a Web event without `Type`, `Message`, or `Config` when the event is
  tied to a config-specific operation.
- Extending deprecated Python Socket.IO events instead of using Go HTTP polling.
