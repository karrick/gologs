GO_TESTS = $(wildcard *_test.go)

.PHONY: bench
bench:
	go test -bench=BenchmarkLog -benchmem

.PHONY: serve
serve: cpuprofile.out
	go tool pprof -http=172.16.20.4:8080 cpuprofile.out

.PHONY: flame
flame: flame.svg

.PHONY: clean
clean:
	rm -rf *.out *.svg

flame.svg: cpuprofile.out
	stackcollapse-go.pl cpuprofile.out | flamegraph.pl > flame.svg

cpuprofile.out: $(GO_TESTS)
	go test -bench=BenchmarkFlamegraph -benchmem -cpuprofile=cpuprofile.out
