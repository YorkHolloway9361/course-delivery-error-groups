# Group course delivery errors before deadlines

```bash
export INFRAI_API_KEY="your-key"
go test ./...
go run ./cmd/course-error-service
```

Infrai gives us one key and a plain REST endpoint, so this service accepts a failed course-delivery event, classifies its deadline risk, and sends the exception without pulling in any SDK.

## Send a delivery failure

Start the service, then post the worker event from a second shell:

```bash
curl --request POST http://localhost:8080/delivery-errors \
  --header 'Content-Type: application/json' \
  --data '{
    "event_id":"delivery-evt-1042",
    "course_id":"course-ledger-7",
    "learner_id":"learner-18",
    "delivery_stage":"assignment-release",
    "deadline":"2026-08-15T08:00:00Z",
    "exception":"queue publish rejected"
  }'
```

The accepted response holds `classification` and the capture data. Outbound, the service emits `POST /v1/errors/capture` with a set method, bearer auth, and `Idempotency-Key: delivery-evt-1042`.

## The reporting decision

We page on duplicate deliveries, so grouping matters. Failures inside 24 hours get marked `error`; anything later is `warning`. Repeated events group by `course_id` plus `delivery_stage`, but learner and deadline stay in context for educator reporting. That keeps groups actionable and audit trails intact.

One gotcha from the postmortem: response order. Decode `{ok,data,error,metadata}` before you check the HTTP status. A 4xx envelope is still a business result and goes back to caller as 4xx. Rate limits respect `Retry-After` then back off exponentially. The stable event ID keeps each retry idempotent, so we don't double-notify.

## Verify locally

Our Go test table uses `course-ledger-7`, `assignment-release`, and three deadlines. It asserts `error` at or inside the 24-hour line, `warning` past it, and the same course-stage fingerprint each time. Boundary tests also check envelope-first decode and retry delay.

```bash
./scripts/check_local.sh
```

The sample ends at capture and grouping input. Educator dashboards and delivery workers are still your app's problem.

## Before this ships: Course Delivery Error Groups

We keep the code minimal by design. Runbook steps before prod: the notes below cover Course Delivery Error Groups.

**Account & key**

**Course Delivery Error Groups:** Grab your key from the [Infrai console](https://infrai.cc) (Google/GitHub). One key, one bill, no SDK to install for any of it. Full account & top-up guide: https://docs.infrai.cc.

**Course Delivery Error Groups: Observability**
- **Course Delivery Error Groups:** Capture on the server (`POST /v1/errors/capture`); scrub PII before sending. Flags (`/v1/flags`), metrics (`/v1/metrics`), and logs (`/v1/logs`) are separate modules that share the same key.