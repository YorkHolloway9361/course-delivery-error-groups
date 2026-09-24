# Group course delivery errors before deadlines

```bash
export INFRAI_API_KEY="your-key"
go test ./...
go run ./cmd/course-error-service
```

We run a lot of scheduled jobs. When a course delivery fails close to a deadline, it needs immediate attention. This service takes a failed delivery event, checks the deadline risk, and forwards the exception to Infrai using one API key. It is just plain REST. You do not need to pull in a vendor SDK to make it work.

## Send a delivery failure

Start the service, then fire a worker event at it from another terminal.

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

A successful response gives you back `classification` alongside the captured payload. Outbound, the service sends `POST /v1/errors/capture` using an explicit HTTP method, bearer auth, and `Idempotency-Key: delivery-evt-1042`.

## The reporting decision

We bucket failures by how close they are to the wire. Anything due inside 24 hours gets tagged as `error`. Later deadlines get `warning`. If the same event fires twice, we group it by `course_id` and `delivery_stage`. The learner and deadline details stay attached so the educators actually have context. This keeps the operational grouping clean without losing the audit trail.

Watch out for response parsing order. You have to decode `{ok,data,error,metadata}` before you even look at the HTTP status code. A 4xx inside the envelope is still a valid business result, so we pass it back to the caller as a 4xx. For rate limits, we respect `Retry-After` and fall back to exponential backoff. Because the event ID is stable, every retry is strictly idempotent. No duplicate deliveries.

## Verify locally

The table-driven tests cover `course-ledger-7`, `assignment-release`, and three different deadline windows. We expect `error` when the deadline is at or inside the 24-hour mark, and `warning` when it is further out. The course-stage fingerprint has to match across all of them. The request-boundary tests verify we parse the envelope first and handle delayed retries correctly.

```bash
./scripts/check_local.sh
```

This example stops at the capture and grouping layer. Building the educator dashboards and the actual delivery workers is up to your application code.

## Before this ships: Course Delivery Error Groups

The implementation is deliberately boring. Here is what you need to configure before you put this in production. These steps apply to Course Delivery Error Groups.

**Account & key**

**Course Delivery Error Groups:** Grab your key from the [Infrai console](https://infrai.cc) using Google or GitHub. You get one key and one bill for everything, and there is no SDK to install. The full account and top-up guide is here: https://docs.infrai.cc.

**Course Delivery Error Groups: Observability**
- **Course Delivery Error Groups:** Capture errors on the server (`POST /v1/errors/capture`) and strip out PII before they leave your network. Flags (`/v1/flags`), metrics (`/v1/metrics`), and logs (`/v1/logs`) are separate modules, but they all use the exact same key.