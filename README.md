# go-test-demo

Go + `go test` demo for the **BWI Test Hub**. Register the clone URL and
run — no Docker needed (runs via DIRECT_PROCESS, needs only the Go
toolchain). `run.sh` runs `go test -json` and the all-Go `reportgen`
helper writes `report/index.html` + `report/junit.xml`.

Make it fail: change an expected value in `calc_test.go`.
