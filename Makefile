ROOT := $(CURDIR)
export GOCACHE := $(ROOT)/.cache/go-build
export GOMODCACHE := $(ROOT)/.cache/go-mod

.PHONY: check test frontend build prototype clean
check: frontend
	npm run check
	cd apps/desktop && go vet ./...
test: frontend
	cd apps/desktop && go test -race ./...
frontend:
	npm run build:frontend
build: frontend
	cd apps/desktop && CGO_LDFLAGS="-framework UniformTypeIdentifiers" go build -tags desktop,production -o build/bin/assessment-caddy .
prototype: frontend
	cd apps/desktop && bash scripts/package-macos.sh
clean:
	rm -rf apps/desktop/frontend/dist apps/desktop/build/bin
