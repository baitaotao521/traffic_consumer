# Error Handling

> Error propagation, API responses, CLI exits, and cancellation behavior.

---

## Overview

The Go backend uses ordinary `error` values and keeps error handling at package
boundaries:

- CLI parsing and execution return errors to `main.go`, which prints to stderr
  and exits with code `2` for parse errors or `1` for runtime errors.
- HTTP handlers translate errors into JSON responses with stable status codes.
- Runtime and consumer code return business errors instead of panicking.
- Background work uses `context.Context`, cancellation functions, and scheduler
  stop hooks.

Do not introduce a custom error framework unless there is a concrete repeated
need. Current code is intentionally simple.

---

## CLI Errors

`main.go` is responsible for process exits:

```go
args, _, err := cli.Parse(os.Args[1:])
if err != nil {
    fmt.Fprintln(os.Stderr, err)
    os.Exit(2)
}
if args.IsCLIMode() {
    if err := cli.Run(args, paths); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

CLI package functions should return errors for invalid config, invalid schedule
parameters, failed storage operations, and malformed header JSON. They should
print success/status messages only after the operation is complete.

Examples:

- `scheduleSpec` returns `未设置有效调度参数` when neither cron nor interval is
  usable.
- `mergeHeaders` returns an error for invalid `--headers-json`.
- `storage.DeleteConfigEntry` returns `(false, nil)` when a config is absent so
  the CLI can print a not-found message without treating it as a system failure.

---

## HTTP Error Responses

HTTP handlers use `writeError`, which writes both `error` and `message` for
frontend compatibility:

```go
func writeError(w http.ResponseWriter, status int, message string) {
    writeJSON(w, status, map[string]string{"error": message, "message": message})
}
```

Current status-code conventions:

- `400 Bad Request`: invalid JSON payload or invalid cron expression.
- `404 Not Found`: missing config or missing plan detail.
- `409 Conflict`: valid request blocked by runtime state, such as starting a
  second download while one is active.
- `500 Internal Server Error`: storage, template, or unexpected runtime failure.

Keep handler methods small: decode request, call `Manager`/`storage`, translate
the result into JSON.

Example from `internal/httpapi/server.go`:

```go
status, err := s.Manager.StartConsumer(cfg)
if err != nil {
    writeError(w, http.StatusConflict, err.Error())
    return
}
writeJSON(w, http.StatusOK, status)
```

---

## Runtime And Scheduler Errors

`runtime.Manager` returns business errors with user-visible Chinese messages
because the frontend displays them directly.

Examples:

- `StartConsumer` returns `流量消耗器已在运行。` when another download is active.
- `StartTempTest` returns a conflict when multiple scheduled plans exist or
  when the scheduled plan belongs to a different config.
- `PlanDetail` maps missing data to an error that the HTTP layer converts to
  `404`.

Scheduler work must be stopped cleanly. Existing CLI scheduler code calls
`scheduler.Stop()` and waits briefly for the stop context before returning.

Scheduled Web runs call `Consumer.RunOnce` synchronously from cron jobs. A
scheduled run must still register a per-run cancel function inside the consumer
so `StopRuntimeConfig` and `StopAll` can interrupt blocked HTTP requests,
thread-start delays, retry backoff, and limiter waits. Do not rely on the
caller passing a cancelable context; Web cron jobs may be triggered with
`context.Background()`.

Temporary tests are exclusive runtime probes. While a temp test is active or
still waiting for cleanup, normal runtime starts and new temp tests must return
a conflict instead of creating another item; otherwise the temp item can race
with the restored scheduled plan and corrupt the user's expected runtime state.

Global stop commands must preserve their visible command scope. The Web
`stop-runtime` command is the "stop all" operation used by the dashboard, but
`stop-scheduler` must stop scheduled runtime items (`Cron != nil`) and paused
plans stored as temp-test resume targets, while leaving ordinary immediate
downloads or temp tests running. If a scheduled item is currently executing a
download round, stopping the scheduler must cancel that active round and record
the run as `已停止`.
Pending temp-test cleanup is still runtime state: `stop-runtime` must clear it
and report success because it cancels a possible scheduled-plan restore.

---

## Consumer Errors

The download loop treats network failures as recoverable until retry exhaustion:

- `downloadWithRetries` retries up to `maxRetries`.
- Each failed attempt is logged once per retry key.
- After the final attempt, `URLMgr.MarkInvalid` marks the URL invalid and emits
  a download failure event.
- When all URLs are invalid, the consumer stops.
- If `AutoRemoveFailedURL` is enabled, the failed URL is removed from both
  runtime state and `config.json`.

Do not hide failed downloads by incrementing success counters. A URL is counted
only after a complete response stream succeeds.

Transient URL invalidation is per run. `URLMgr.ResetRuntimeState` must clear
temporary invalid flags and weights before the next scheduled round, while
`AutoRemoveFailedURL` remains persistent because it removes the URL from the
runtime list and saved config. Retry loops should check whether another thread
already marked the same URL invalid before starting or continuing backoff work.

Business stop paths should cancel the active run context, not only flip an
`active` boolean. User stop, duration limit, traffic limit, count limit, no
available URL, and all-URL-invalid stops all need to unblock peer goroutines.

---

## Storage Errors

Preserve the current compatibility behavior:

- Missing JSON files are treated as empty data.
- Empty JSON files are treated as empty data.
- Corrupt JSON is currently treated as empty data, matching the legacy Python
  tolerance.
- Write failures return errors; business code must not swallow them.

The one intentional panic is `storage.MustWriteJSON`, which exists for test
helpers only. Do not use it in runtime code.

---

## Common Mistakes

- Returning plain text from API handlers instead of the `{"error","message"}`
  JSON shape.
- Treating scheduler conflicts as `500` instead of `409`.
- Checking only `Consumer.IsActive()` or `Cron != nil` when deciding whether
  runtime state exists. Temporary-test cleanup items can be inactive and have no
  cron while still carrying a `ResumeTarget` that would restore a scheduled
  plan.
- Logging an error without returning or surfacing it to the API/CLI caller.
- Adding Python-side exception handling for new behavior. Python is deprecated;
  fix or preserve behavior in Go.
