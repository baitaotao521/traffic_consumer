# 修复调度与下载流程问题 - Design

## Boundaries

- Backend-only change under `internal/runtime`, `internal/consumer`, and `internal/urlmanager`.
- No Web API schema change is planned; existing status payload fields keep their JSON shape.
- `记录文档.md` lives at repository root and records each completed fix.

## Runtime Cancellation

`Consumer.RunOnce` will create a per-run child context from the caller context and publish that run cancel function on `Consumer.cancel`. `Stop()` will then cancel both async and scheduled synchronous runs. `StartAsync` keeps returning its existing `done` channel, so external callers do not lose completion signaling.

The per-run cancel is cleared only if it still belongs to the finishing run. This avoids an older run clearing a newer run's cancel function.

## URL Runtime State

`URLMgr.ResetRuntimeState` resets per-round thread status, thread metrics, usage counts, transient invalid flags, and weights. Runtime URL removal remains persistent because removed URLs are no longer present in `m.urls`. This preserves `AutoRemoveFailedURL` semantics while letting transient failures recover on later scheduled rounds.

`URLMgr.IsInvalid(url)` provides a safe read path for retry loops. `Consumer.downloadWithRetries` checks it before each attempt and after retry backoff so concurrent invalidation stops redundant work.

## Stop Responsiveness

The rate limiter gains a context-aware acquire path. Download streaming will use the context-aware method so stop/cancel does not wait for the limiter's sleep loop. The old `Acquire(numBytes)` API stays as a compatibility wrapper for tests and future callers.

## Run Result Semantics

`Consumer` stores a protected `finishReason`. Business stop paths set a specific reason near the triggering condition:

- user stop: `已停止`
- traffic limit: `达到流量限制`
- count limit: `达到下载次数限制`
- duration limit: `达到时长限制`
- no available URL: `无可用链接`
- all URLs invalid: `全部链接失效`

`finishRun` writes that reason to memory history and `stats.json`. Natural completion keeps `成功`.

## Multi-Config URL Usage

`Manager.Status` will aggregate URL usage across every active item. Duplicate URLs remain displayed once, with counts summed. This matches the existing multi-config thread aggregation behavior and does not change the payload schema.

## Compatibility

- Existing config and stats files remain readable.
- Existing tests expecting `RateLimiter.Acquire` continue to compile.
- Existing UI consumers still receive `[]URLUsageStat`.

## Rollback

Each change is isolated:

- Cancellation changes are limited to `Consumer.RunOnce`, `Stop`, and limiter call sites.
- URL reset and invalid-short-circuit changes are limited to `urlmanager` and retry loop calls.
- Result reason changes are limited to `consumer`.
- URL usage aggregation is limited to `runtime`.

If a regression appears, revert the relevant isolated section and its tests.
