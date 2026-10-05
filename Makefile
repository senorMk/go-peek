ROOT := $(CURDIR)
export GOCACHE := $(ROOT)/.cache/go-build
export GOMODCACHE := $(ROOT)/.cache/go-mod

.PHONY: check test frontend build prototype capture-check capture-video capture-external setup-key clean
check: frontend
	npm run check
	cd apps/desktop && go vet ./...
test: frontend
	cd apps/desktop && go test -race ./...
frontend:
	npm run build:frontend
build: frontend
	cd apps/desktop && CGO_LDFLAGS="-framework UniformTypeIdentifiers" go build -tags desktop,production -o build/bin/go-peek .
prototype: frontend
	cd apps/desktop && bash scripts/package-macos.sh
capture-check:
	bash apps/desktop/scripts/capture-check.sh screenshot
capture-video:
	bash apps/desktop/scripts/capture-check.sh video
capture-external:
	bash apps/desktop/scripts/capture-check.sh "$(TOOL)" "$(or $(MODE),display)"
setup-key:
	bash apps/desktop/scripts/setup-key.sh "$(PROVIDER)"
clean:
	rm -rf apps/desktop/frontend/dist apps/desktop/build/bin
