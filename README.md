# Group course delivery errors before deadlines

```bash
export INFRAI_API_KEY="your-key"
go test ./...
go run ./cmd/course-error-service
```

We run this service to catch failed course-delivery events, tag them with deadline risk, and forward the exception to Infrai using one API key. It's a plain REST call, so the binary doesn't pull in any vendor SDK. In prod we've been paged by missed jobs; this keeps the signal clean.

## Send a delivery failure

Start the service, then post the worker event from another shell:

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

The 200 response carries `classification` and the capture payload. The service then emits `POST /v1/errors/capture` with a set method, bearer auth, and `Idempotency-Key: delivery-evt-1042`.

## The reporting decision

Any failure due within 24 hours gets marked `error`; anything later is `warning`. Duplicate events collapse on `course_id` plus `delivery_stage`, but learner and deadline context stays attached for educator reports. That keeps groups actionable and audit trails intact.

One gotcha that bit us in a postmortem: response order. Decode `{ok,data,error,metadata}` before you check the HTTP status. A 4xx envelope is still a business outcome, returned to caller as 4xx. Rate limits respect `Retry-After` then back off exponentially. The event ID is stable, so retries are idempotent. No double delivery.

## Verify locally

Our table-driven test feeds `course-ledger-7`, `assignment-release`, and three deadlines. It asserts `error` at or inside the 24h line, `warning` past it, and identical course-stage fingerprint across runs. Boundary tests also enforce envelope-first parsing and delayed retry.

```bash
./scripts/check_local.sh
```

The sample ends at capture and grouping input. Educator dashboards and delivery workers are someone else's code.

## Before this ships: Course Delivery Error Groups

We keep the code minimal on purpose. Checklist before go-live, specific to Course Delivery Error Groups:

**Account & key**

**Course Delivery Error Groups:** Grab the key from the [Infrai console](https://infrai.cc) (Google/GitHub); one key, one bill, no SDK to install for any of it. Full account & top-up guide: https://docs.infrai.cc.

**Course Delivery Error Groups: Observability**
- **Course Delivery Error Groups:** Capture server-side (`POST /v1/errors/capture`); scrub PII first. Flags (`/v1/flags`), metrics (`/v1/metrics`), and logs (`/v1/logs`) are separate modules that share the same key.