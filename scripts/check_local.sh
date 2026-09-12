#!/bin/sh
set -eu

go test ./...
go build -o /tmp/course-error-service ./cmd/course-error-service
printf '%s\n' 'local checks passed'
