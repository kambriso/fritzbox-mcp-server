.POSIX:

TAG := $(shell git describe --tags --always --dirty)
VERSION := $(shell git describe --tags --always --dirty | sed 's/^v//')
COMMIT := $(shell git rev-parse --short HEAD)

# Reproducible builds
SOURCE_DATE_EPOCH ?= $(shell git log -1 --format=%ct)
export SOURCE_DATE_EPOCH

GO ?= CGO_ENABLED=0 go

BINARY = fritz-mcp
MAIN = .
BUILD_DIR ?= build
DIST_DIR ?= dist
LDFLAGS = -X main.version=$(VERSION) -X main.commit=$(COMMIT) -buildid=
DISTFLAGS = -trimpath -buildvcs=false -ldflags "-s -w -extldflags=-static $(LDFLAGS)"
STRIP ?= strip
GO_SOURCES = $(wildcard **/*.go)

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*##' $(MAKEFILE_LIST) | awk -F ':.*## ' '{printf "  %-42s %s\n", $$1, $$2}'

.PHONY: build
build: ## Build the binary
	$(GO) build -o $(BINARY) $(MAIN)

.PHONY: run
run: ## Run without building (use ARGS for arguments)
	$(GO) run $(MAIN) $(ARGS)

.PHONY: test
test: ## Run tests
	$(GO) test -v ./...

.PHONY: clean
clean: ## Remove build artifacts
	rm -f $(BINARY)
	rm -rf $(BUILD_DIR)/ $(DIST_DIR)/
	$(GO) clean

.PHONY: install
install: ## Install to GOPATH/bin
	$(GO) install $(DISTFLAGS) $(MAIN)

.PHONY: tidy
tidy: ## Format, vet, and lint
	$(GO) fmt ./...
	$(GO) vet ./...
	$(GO) run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run

.PHONY: lint
lint: vet fmt

.PHONY: check
check: lint test ## Run lint and tests

$(BUILD_DIR)/linux/amd64/$(BINARY): $(GO_SOURCES)
	mkdir -p $(dir $@)
	GOOS=linux GOARCH=amd64 $(GO) build $(DISTFLAGS) -o $@ $(MAIN)
	$(STRIP) --strip-all $@ 2>/dev/null || true

$(BUILD_DIR)/linux/arm64/$(BINARY): $(GO_SOURCES)
	mkdir -p $(dir $@)
	GOOS=linux GOARCH=arm64 $(GO) build $(DISTFLAGS) -o $@ $(MAIN)
	$(STRIP) --strip-all $@ 2>/dev/null || true

$(BUILD_DIR)/linux/386/$(BINARY): $(GO_SOURCES)
	mkdir -p $(dir $@)
	GOOS=linux GOARCH=386 $(GO) build $(DISTFLAGS) -o $@ $(MAIN)
	$(STRIP) --strip-all $@ 2>/dev/null || true

$(BUILD_DIR)/linux/arm/$(BINARY): $(GO_SOURCES)
	mkdir -p $(dir $@)
	GOOS=linux GOARCH=arm $(GO) build $(DISTFLAGS) -o $@ $(MAIN)
	$(STRIP) --strip-all $@ 2>/dev/null || true

$(BUILD_DIR)/darwin/amd64/$(BINARY): $(GO_SOURCES)
	mkdir -p $(dir $@)
	GOOS=darwin GOARCH=amd64 $(GO) build $(DISTFLAGS) -o $@ $(MAIN)
	$(STRIP) --strip-all $@ 2>/dev/null || true

$(BUILD_DIR)/darwin/arm64/$(BINARY): $(GO_SOURCES)
	mkdir -p $(dir $@)
	GOOS=darwin GOARCH=arm64 $(GO) build $(DISTFLAGS) -o $@ $(MAIN)
	$(STRIP) --strip-all $@ 2>/dev/null || true

$(BUILD_DIR)/windows/amd64/$(BINARY).exe: $(GO_SOURCES)
	mkdir -p $(dir $@)
	GOOS=windows GOARCH=amd64 $(GO) build $(DISTFLAGS) -o $@ $(MAIN)

$(BUILD_DIR)/windows/arm64/$(BINARY).exe: $(GO_SOURCES)
	mkdir -p $(dir $@)
	GOOS=windows GOARCH=arm64 $(GO) build $(DISTFLAGS) -o $@ $(MAIN)

$(BUILD_DIR)/wasi/wasm/$(BINARY).wasm: $(GO_SOURCES)
	mkdir -p $(dir $@)
	GOOS=wasip1 GOARCH=wasm $(GO) build $(DISTFLAGS) -o $@ $(MAIN)

$(DIST_DIR):
	mkdir -p $@

DIST_BINARIES = \
	$(BUILD_DIR)/linux/amd64/$(BINARY) \
	$(BUILD_DIR)/linux/arm64/$(BINARY) \
	$(BUILD_DIR)/linux/386/$(BINARY) \
	$(BUILD_DIR)/linux/arm/$(BINARY) \
	$(BUILD_DIR)/darwin/amd64/$(BINARY) \
	$(BUILD_DIR)/darwin/arm64/$(BINARY) \
	$(BUILD_DIR)/windows/amd64/$(BINARY).exe \
	$(BUILD_DIR)/windows/arm64/$(BINARY).exe \
	$(BUILD_DIR)/wasi/wasm/$(BINARY).wasm

DIST_ARCHIVES = \
	$(DIST_DIR)/$(BINARY)-linux-amd64.tar.xz \
	$(DIST_DIR)/$(BINARY)-linux-arm64.tar.xz \
	$(DIST_DIR)/$(BINARY)-linux-386.tar.xz \
	$(DIST_DIR)/$(BINARY)-linux-arm.tar.xz \
	$(DIST_DIR)/$(BINARY)-darwin-amd64.tar.xz \
	$(DIST_DIR)/$(BINARY)-darwin-arm64.tar.xz \
	$(DIST_DIR)/$(BINARY)-windows-amd64.zip \
	$(DIST_DIR)/$(BINARY)-windows-arm64.zip \
	$(DIST_DIR)/$(BINARY)-wasi.tar.xz

$(DIST_DIR)/$(BINARY)-linux-amd64.tar.xz: $(BUILD_DIR)/linux/amd64/$(BINARY) | $(DIST_DIR)
	tar -cJf $@ -C $(BUILD_DIR)/linux/amd64 $(BINARY)

$(DIST_DIR)/$(BINARY)-linux-arm64.tar.xz: $(BUILD_DIR)/linux/arm64/$(BINARY) | $(DIST_DIR)
	tar -cJf $@ -C $(BUILD_DIR)/linux/arm64 $(BINARY)

$(DIST_DIR)/$(BINARY)-linux-386.tar.xz: $(BUILD_DIR)/linux/386/$(BINARY) | $(DIST_DIR)
	tar -cJf $@ -C $(BUILD_DIR)/linux/386 $(BINARY)

$(DIST_DIR)/$(BINARY)-linux-arm.tar.xz: $(BUILD_DIR)/linux/arm/$(BINARY) | $(DIST_DIR)
	tar -cJf $@ -C $(BUILD_DIR)/linux/arm $(BINARY)

$(DIST_DIR)/$(BINARY)-darwin-amd64.tar.xz: $(BUILD_DIR)/darwin/amd64/$(BINARY) | $(DIST_DIR)
	tar -cJf $@ -C $(BUILD_DIR)/darwin/amd64 $(BINARY)

$(DIST_DIR)/$(BINARY)-darwin-arm64.tar.xz: $(BUILD_DIR)/darwin/arm64/$(BINARY) | $(DIST_DIR)
	tar -cJf $@ -C $(BUILD_DIR)/darwin/arm64 $(BINARY)

$(DIST_DIR)/$(BINARY)-windows-amd64.zip: $(BUILD_DIR)/windows/amd64/$(BINARY).exe | $(DIST_DIR)
	cd $(BUILD_DIR)/windows/amd64 && zip -q $(CURDIR)/$@ $(BINARY).exe

$(DIST_DIR)/$(BINARY)-windows-arm64.zip: $(BUILD_DIR)/windows/arm64/$(BINARY).exe | $(DIST_DIR)
	cd $(BUILD_DIR)/windows/arm64 && zip -q $(CURDIR)/$@ $(BINARY).exe

$(DIST_DIR)/$(BINARY)-wasi.tar.xz: $(BUILD_DIR)/wasi/wasm/$(BINARY).wasm | $(DIST_DIR)
	tar -cJf $@ -C $(BUILD_DIR)/wasi/wasm $(BINARY).wasm

.PHONY: dists
dists: $(DIST_BINARIES) ## Build all platform binaries

.PHONY: dists-container
dists-container: ## Build all platform binaries in golang:1.25 container for reproducibility
	podman run --rm -v .:/work:z -w /work golang:1.25 make dists

.PHONY: archives
archives: $(DIST_ARCHIVES) ## Create release archives

.PHONY: sbom
sbom: $(DIST_DIR)/sbom.json

$(DIST_DIR)/sbom.json: go.mod go.sum | $(DIST_DIR)
	$(GO) run github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@latest app -json -output $@ -main $(MAIN)

$(DIST_DIR)/SHA256SUMS: $(DIST_ARCHIVES)
	cd $(DIST_DIR) && sha256sum $(BINARY)-*.tar.xz $(BINARY)-*.zip > SHA256SUMS

.PHONY: checksums
checksums: $(DIST_DIR)/SHA256SUMS ## Generate SHA256 checksums for all platforms (use make -j for parallel)

.PHONY: mcpb
mcpb: $(DIST_DIR)/fritzbox-mcp-server.mcpb ## Build Claude Desktop Extension (.mcpb package)

.PHONY: all
all: archives checksums mcpb sbom ## Build all distribution artifacts

MCPB_STAGING_DIR = $(BUILD_DIR)/mcpb-staging

$(MCPB_STAGING_DIR)/bin/linux-amd64/$(BINARY): $(BUILD_DIR)/linux/amd64/$(BINARY)
	install -D $< $@

$(MCPB_STAGING_DIR)/bin/linux-arm64/$(BINARY): $(BUILD_DIR)/linux/arm64/$(BINARY)
	install -D $< $@

$(MCPB_STAGING_DIR)/bin/darwin-amd64/$(BINARY): $(BUILD_DIR)/darwin/amd64/$(BINARY)
	install -D $< $@

$(MCPB_STAGING_DIR)/bin/darwin-arm64/$(BINARY): $(BUILD_DIR)/darwin/arm64/$(BINARY)
	install -D $< $@

$(MCPB_STAGING_DIR)/bin/windows-amd64/$(BINARY).exe: $(BUILD_DIR)/windows/amd64/$(BINARY).exe
	install -D $< $@

$(MCPB_STAGING_DIR)/bin/windows-arm64/$(BINARY).exe: $(BUILD_DIR)/windows/arm64/$(BINARY).exe
	install -D $< $@

$(MCPB_STAGING_DIR)/manifest.json: mcpb/manifest.json
	mkdir -p $(dir $@)
	jq '.version = "$(VERSION)"' $< > $@

MCPB_STAGED_FILES = \
	$(MCPB_STAGING_DIR)/bin/linux-amd64/$(BINARY) \
	$(MCPB_STAGING_DIR)/bin/linux-arm64/$(BINARY) \
	$(MCPB_STAGING_DIR)/bin/darwin-amd64/$(BINARY) \
	$(MCPB_STAGING_DIR)/bin/darwin-arm64/$(BINARY) \
	$(MCPB_STAGING_DIR)/bin/windows-amd64/$(BINARY).exe \
	$(MCPB_STAGING_DIR)/bin/windows-arm64/$(BINARY).exe \
	$(MCPB_STAGING_DIR)/manifest.json

$(DIST_DIR)/fritzbox-mcp-server.mcpb: $(MCPB_STAGED_FILES) | $(DIST_DIR)
	cd $(MCPB_STAGING_DIR) && zip -qr $(CURDIR)/$@ .

.PHONY: release
release: $(DIST_DIR)/sbom.json checksums ## Create GitHub release (local use only)
	gh release create "$(VERSION)" \
		--repo kambriso/fritzbox-mcp-server \
		--target bdbb079c98e7c7fd9ac3b1a7d1a09a777a1e236f \
		--title "Release $(VERSION)" \
		--notes "Release $(VERSION)" \
		$(DIST_DIR)/$(BINARY)-linux-amd64.tar.xz \
		$(DIST_DIR)/$(BINARY)-linux-arm64.tar.xz \
		$(DIST_DIR)/$(BINARY)-linux-386.tar.xz \
		$(DIST_DIR)/$(BINARY)-linux-arm.tar.xz \
		$(DIST_DIR)/$(BINARY)-darwin-amd64.tar.xz \
		$(DIST_DIR)/$(BINARY)-darwin-arm64.tar.xz \
		$(DIST_DIR)/$(BINARY)-windows-amd64.zip \
		$(DIST_DIR)/$(BINARY)-windows-arm64.zip \
		$(DIST_DIR)/$(BINARY)-wasi.tar.xz \
		$(DIST_DIR)/sbom.json \
		$(DIST_DIR)/SHA256SUMS

.PHONY: container-login
container-login: ## Login to GitHub Container Registry using gh
	gh auth token | buildah login ghcr.io -u kambriso --password-stdin

.PHONY: container-wasi
container-wasi: $(BUILD_DIR)/wasi/wasm/$(BINARY).wasm ## Build WASI container image
	buildah bud \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg DATE=$(shell git log -1 --format=%cI) \
		-t ghcr.io/kambriso/fritzbox-mcp-server:$(VERSION)-wasi \
		-f Containerfile.wasi .

.PHONY: container-multiarch
container-multiarch: $(BUILD_DIR)/linux/amd64/$(BINARY) $(BUILD_DIR)/linux/arm64/$(BINARY) ## Build multi-arch container image
	buildah manifest rm ghcr.io/kambriso/fritzbox-mcp-server:$(VERSION) 2>/dev/null || true
	buildah manifest create ghcr.io/kambriso/fritzbox-mcp-server:$(VERSION)
	buildah bud --platform linux/amd64 \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg DATE=$(shell git log -1 --format=%cI) \
		--manifest ghcr.io/kambriso/fritzbox-mcp-server:$(VERSION) \
		-f Containerfile .
	buildah bud --platform linux/arm64 \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg DATE=$(shell git log -1 --format=%cI) \
		--manifest ghcr.io/kambriso/fritzbox-mcp-server:$(VERSION) \
		-f Containerfile .

.PHONY: container-push-wasi
container-push-wasi: container-wasi ## Push WASI container to ghcr.io
	buildah push ghcr.io/kambriso/fritzbox-mcp-server:$(VERSION)-wasi

.PHONY: container-push-multiarch
container-push-multiarch: container-multiarch ## Push multi-arch container to ghcr.io
	buildah manifest push --all ghcr.io/kambriso/fritzbox-mcp-server:$(VERSION) docker://ghcr.io/kambriso/fritzbox-mcp-server:$(VERSION)

.PHONY: containers
containers: container-wasi container-multiarch ## Build all containers

.PHONY: containers-push
containers-push: container-push-wasi container-push-multiarch ## Push all containers to ghcr.io

.PHONY: run-wasi
run-wasi: $(BUILD_DIR)/wasi/wasm/$(BINARY).wasm ## Run WASI binary with wasmedge (requires .env, use ARGS for arguments)
	bash -c 'set -a; [ -f .env ] && . ./.env || [ -f ~/.config/fritzbox-mcp-server/.env ] && . ~/.config/fritzbox-mcp-server/.env; wasmedge --dir .:. --env FRITZ_HOST=$$FRITZ_HOST --env FRITZ_PORT=$$FRITZ_PORT --env FRITZ_USERNAME=$$FRITZ_USERNAME --env FRITZ_PASSWORD=$$FRITZ_PASSWORD --env FRITZ_TLS=$$FRITZ_TLS $< $(ARGS)'

.PHONY: run-container
run-container: ## Run published multi-arch container from ghcr.io (requires .env, use ARGS for arguments)
	bash -c 'set -a; [ -f .env ] && . ./.env || [ -f ~/.config/fritzbox-mcp-server/.env ] && . ~/.config/fritzbox-mcp-server/.env; \
	podman run --rm \
		--env FRITZ_HOST --env FRITZ_PORT --env FRITZ_USERNAME --env FRITZ_PASSWORD --env FRITZ_TLS \
		ghcr.io/kambriso/fritzbox-mcp-server:$(VERSION) $(ARGS)'

.PHONY: run-container-local
run-container-local: container-multiarch ## Build and run local multi-arch container (requires .env, use ARGS for arguments)
	bash -c 'set -a; [ -f .env ] && . ./.env || [ -f ~/.config/fritzbox-mcp-server/.env ] && . ~/.config/fritzbox-mcp-server/.env; \
	podman run --rm \
		--env FRITZ_HOST --env FRITZ_PORT --env FRITZ_USERNAME --env FRITZ_PASSWORD --env FRITZ_TLS \
		ghcr.io/kambriso/fritzbox-mcp-server:$(VERSION) $(ARGS)'

.PHONY: run-wasi-container
run-wasi-container: container-wasi ## Run WASI OCI container with podman (use ARGS for arguments)
	podman run --rm --runtime=crun --annotation=module.wasm.image/variant=compat \
		--env FRITZ_HOST --env FRITZ_PORT --env FRITZ_USERNAME --env FRITZ_PASSWORD --env FRITZ_TLS \
		ghcr.io/kambriso/fritzbox-mcp-server:$(VERSION)-wasi $(ARGS)

.PHONY: deps
deps: ## List dependencies
	$(GO) list -m all

# SLSA verification
GITHUB_REPO ?= kambriso/fritzbox-mcp-server
GITLAB_REPO ?= kambriso/fritzbox-mcp-server
VERIFY_DIR ?= $(BUILD_DIR)/verify

# GitHub SLSA Level 3 verification (downloads binary + provenance, then verifies)
define slsa-verify-github
	mkdir -p $(VERIFY_DIR)/github
	gh release download $(TAG) -R $(GITHUB_REPO) -D $(VERIFY_DIR)/github --clobber \
		-p '$(1)' -p 'multiple.intoto.jsonl'
	go run github.com/slsa-framework/slsa-verifier/v2/cli/slsa-verifier@latest verify-artifact \
		$(VERIFY_DIR)/github/$(1) \
		--provenance-path $(VERIFY_DIR)/github/multiple.intoto.jsonl \
		--source-uri github.com/$(GITHUB_REPO)
endef

.PHONY: verify-slsa-github-linux-amd64
verify-slsa-github-linux-amd64: ## Verify GitHub linux-amd64
	$(call slsa-verify-github,fritz-mcp-linux-amd64)

.PHONY: verify-slsa-github-linux-arm64
verify-slsa-github-linux-arm64: ## Verify GitHub linux-arm64
	$(call slsa-verify-github,fritz-mcp-linux-arm64)

.PHONY: verify-slsa-github-linux-arm
verify-slsa-github-linux-arm: ## Verify GitHub linux-arm
	$(call slsa-verify-github,fritz-mcp-linux-arm)

.PHONY: verify-slsa-github-linux-386
verify-slsa-github-linux-386: ## Verify GitHub linux-386
	$(call slsa-verify-github,fritz-mcp-linux-386)

.PHONY: verify-slsa-github-darwin-amd64
verify-slsa-github-darwin-amd64: ## Verify GitHub darwin-amd64
	$(call slsa-verify-github,fritz-mcp-darwin-amd64)

.PHONY: verify-slsa-github-darwin-arm64
verify-slsa-github-darwin-arm64: ## Verify GitHub darwin-arm64
	$(call slsa-verify-github,fritz-mcp-darwin-arm64)

.PHONY: verify-slsa-github-windows-amd64
verify-slsa-github-windows-amd64: ## Verify GitHub windows-amd64
	$(call slsa-verify-github,fritz-mcp-windows-amd64.exe)

.PHONY: verify-slsa-github-windows-arm64
verify-slsa-github-windows-arm64: ## Verify GitHub windows-arm64
	$(call slsa-verify-github,fritz-mcp-windows-arm64.exe)

.PHONY: verify-slsa-github-wasi
verify-slsa-github-wasi: ## Verify GitHub wasi
	$(call slsa-verify-github,fritz-mcp-wasi.wasm)

.PHONY: verify-slsa-github
verify-slsa-github: verify-slsa-github-linux-amd64 verify-slsa-github-linux-arm64 verify-slsa-github-linux-arm verify-slsa-github-linux-386 verify-slsa-github-darwin-amd64 verify-slsa-github-darwin-arm64 verify-slsa-github-windows-amd64 verify-slsa-github-windows-arm64 verify-slsa-github-wasi ## Verify GitHub SLSA Level 3 provenance (use TAG=v0.x.x)

# GitLab SLSA Level 2 verification (downloads binary + sig + cert, then verifies)
define slsa-verify-gitlab
	mkdir -p $(VERIFY_DIR)/gitlab
	glab release download $(TAG) -R $(GITLAB_REPO) -D $(VERIFY_DIR)/gitlab \
		-n '$(1)' -n '$(1).sig' -n '$(1).pem'
	go run github.com/sigstore/cosign/v2/cmd/cosign@latest verify-blob \
		--signature $(VERIFY_DIR)/gitlab/$(1).sig \
		--certificate $(VERIFY_DIR)/gitlab/$(1).pem \
		--certificate-identity-regexp '.*' \
		--certificate-oidc-issuer https://gitlab.com \
		$(VERIFY_DIR)/gitlab/$(1)
endef

.PHONY: verify-slsa-gitlab-linux-amd64
verify-slsa-gitlab-linux-amd64: ## Verify GitLab linux-amd64
	$(call slsa-verify-gitlab,fritz-mcp-linux-amd64)

.PHONY: verify-slsa-gitlab-linux-arm64
verify-slsa-gitlab-linux-arm64: ## Verify GitLab linux-arm64
	$(call slsa-verify-gitlab,fritz-mcp-linux-arm64)

.PHONY: verify-slsa-gitlab-linux-arm
verify-slsa-gitlab-linux-arm: ## Verify GitLab linux-arm
	$(call slsa-verify-gitlab,fritz-mcp-linux-arm)

.PHONY: verify-slsa-gitlab-linux-386
verify-slsa-gitlab-linux-386: ## Verify GitLab linux-386
	$(call slsa-verify-gitlab,fritz-mcp-linux-386)

.PHONY: verify-slsa-gitlab-darwin-amd64
verify-slsa-gitlab-darwin-amd64: ## Verify GitLab darwin-amd64
	$(call slsa-verify-gitlab,fritz-mcp-darwin-amd64)

.PHONY: verify-slsa-gitlab-darwin-arm64
verify-slsa-gitlab-darwin-arm64: ## Verify GitLab darwin-arm64
	$(call slsa-verify-gitlab,fritz-mcp-darwin-arm64)

.PHONY: verify-slsa-gitlab-windows-amd64
verify-slsa-gitlab-windows-amd64: ## Verify GitLab windows-amd64
	$(call slsa-verify-gitlab,fritz-mcp-windows-amd64.exe)

.PHONY: verify-slsa-gitlab-windows-arm64
verify-slsa-gitlab-windows-arm64: ## Verify GitLab windows-arm64
	$(call slsa-verify-gitlab,fritz-mcp-windows-arm64.exe)

.PHONY: verify-slsa-gitlab-wasi
verify-slsa-gitlab-wasi: ## Verify GitLab wasi
	$(call slsa-verify-gitlab,fritz-mcp-wasi.wasm)

.PHONY: verify-slsa-gitlab
verify-slsa-gitlab: verify-slsa-gitlab-linux-amd64 verify-slsa-gitlab-linux-arm64 verify-slsa-gitlab-linux-arm verify-slsa-gitlab-linux-386 verify-slsa-gitlab-darwin-amd64 verify-slsa-gitlab-darwin-arm64 verify-slsa-gitlab-windows-amd64 verify-slsa-gitlab-windows-arm64 verify-slsa-gitlab-wasi ## Verify GitLab SLSA Level 2 signatures (use TAG=v0.x.x)

.PHONY: verify-slsa
verify-slsa: verify-slsa-github verify-slsa-gitlab ## Verify SLSA provenance on both forges

# VirusTotal scanning (requires VT_API_KEY environment variable)
define vt-scan
	@echo "Scanning $(1)..."
	curl -s --request POST \
		--url https://www.virustotal.com/api/v3/files \
		--header "x-apikey: $(VT_API_KEY)" \
		--form file=@"$(1)" > /dev/null
	@echo "$(notdir $(1)): https://www.virustotal.com/gui/file/$$(sha256sum $(1) | cut -d' ' -f1)" >> VT_RESULTS.txt
endef

.PHONY: virustotal
virustotal: virustotal-linux-amd64 virustotal-linux-arm64 virustotal-linux-386 virustotal-linux-arm virustotal-darwin-amd64 virustotal-darwin-arm64 virustotal-windows-amd64 virustotal-windows-arm64 virustotal-wasi-wasm ## Scan all binaries with VirusTotal
	cat VT_RESULTS.txt

.PHONY: virustotal-linux-amd64
virustotal-linux-amd64: $(BUILD_DIR)/linux/amd64/$(BINARY)
	$(call vt-scan,$<)

.PHONY: virustotal-linux-arm64
virustotal-linux-arm64: $(BUILD_DIR)/linux/arm64/$(BINARY)
	$(call vt-scan,$<)

.PHONY: virustotal-linux-386
virustotal-linux-386: $(BUILD_DIR)/linux/386/$(BINARY)
	$(call vt-scan,$<)

.PHONY: virustotal-linux-arm
virustotal-linux-arm: $(BUILD_DIR)/linux/arm/$(BINARY)
	$(call vt-scan,$<)

.PHONY: virustotal-darwin-amd64
virustotal-darwin-amd64: $(BUILD_DIR)/darwin/amd64/$(BINARY)
	$(call vt-scan,$<)

.PHONY: virustotal-darwin-arm64
virustotal-darwin-arm64: $(BUILD_DIR)/darwin/arm64/$(BINARY)
	$(call vt-scan,$<)

.PHONY: virustotal-windows-amd64
virustotal-windows-amd64: $(BUILD_DIR)/windows/amd64/$(BINARY).exe
	$(call vt-scan,$<)

.PHONY: virustotal-windows-arm64
virustotal-windows-arm64: $(BUILD_DIR)/windows/arm64/$(BINARY).exe
	$(call vt-scan,$<)

.PHONY: virustotal-wasi-wasm
virustotal-wasi-wasm: $(BUILD_DIR)/wasi/wasm/$(BINARY).wasm
	$(call vt-scan,$<)
