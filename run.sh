#!/bin/sh
# Runs `go test` and converts its JSON output to report/junit.xml +
# report/index.html via the all-Go reportgen helper. `|| true` keeps a
# failing test suite from aborting before the report is generated;
# reportgen carries the real exit code.
set -u
mkdir -p report
go test -json . > report/gotest.json || true
go run ./reportgen < report/gotest.json
