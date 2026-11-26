.PHONY: help install build build-linux enclave lint fix test check menubar-check menubar window catalog catalog-open catalog-export catalog-gallery e2e-machine e2e-binaries provision deprovision generate clean

BINARY       := station
BUILD_PATH   := .artifacts/go/$(BINARY)
LINUX_PATH   := .artifacts/go/$(BINARY)-linux-arm64
SWAP_PATH    := .artifacts/go/$(BINARY)-swap
# value: the binary the CLI specs run, built once before the runner starts, so its
# workers never link it concurrently with the sessions they drive. It is the spec
# build (`-tags world`): with $STATION_WORLD set it runs over the fake world of
# specs/usecases/, without it over the real adapters, as the release build does.
TEST_PATH    := .artifacts/go/$(BINARY)-test
SPEC_TAGS    := world
BIN_DIR      := $(HOME)/.local/bin
INSTALL_PATH := $(BIN_DIR)/$(BINARY)
TURBO        := npx turbo
MENUBAR_APP  := .artifacts/menubar/Station.app
# value: /usr/bin/make on macOS is itself an xcrun shim, so a missing SDK is
# staged by overriding the probe (`make menubar-check SDK_PROBE=false`).
SDK_PROBE    ?= xcrun --sdk macosx --show-sdk-path
WINDOW_APP   := .artifacts/window/Spwn.app
WINDOW_CARGO := .artifacts/window/cargo
WINDOW_ICON  := .artifacts/window/icon
# value: the engine Spwn.app carries (ADR-073, mode A): ENGINE=<path> names a darwin-arm64 build, the release's; without it the target builds one
WINDOW_ENGINE := $(WINDOW_APP)/Contents/MacOS/Spwn Engine
WINDOW_MENUBAR := $(WINDOW_APP)/Contents/Library/LoginItems/Station.app
ENGINE       ?=
ENGINE_VERSION ?= $(shell git describe --tags --always --dirty)
# value: the window's version is the release's, as git describes it for the binaries and Station.app, without its v;
# value: it is compiled into the binary, where the updater reads it, and Tauri writes the bundle's from it
# hazard: a bare hash (a clone with no tag reachable) is no semver, and tauri build refuses it
WINDOW_VERSION ?= $(patsubst v%,%,$(shell git describe --tags --always --dirty))
CATALOG_BIN  := .artifacts/menubar/build/debug/StationCatalog
CATALOG_OUT  ?= .artifacts/catalog
LDFLAGS      := -X github.com/jterrazz/studio-station/apps/engine/wire.version=$(shell git describe --tags --always --dirty)
# value: the Secure Enclave's Swift half, which every darwin build with cgo
# links (CryptoKit has no C API); built for both release arches, from the
# release's macOS floor.
ENCLAVE_LIB  := .artifacts/enclave/libstationenclave.a
ENCLAVE_SRC  := apps/engine/platform/darwin/enclave.swift
MACOS_FLOOR  := 14.0
SPEC_LDFLAGS := -X github.com/jterrazz/studio-station/apps/engine/wire.version=$(shell git describe --tags --always --dirty --long)

CYAN  := \033[36m
GREEN := \033[32m
DIM   := \033[2m
RESET := \033[0m

help: ## Show available targets
	@printf "$(CYAN)studio-station$(RESET)\n"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "$(CYAN)%-12s$(RESET) %s\n", $$1, $$2}'

node_modules/.install: package.json package-lock.json $(wildcard .artifacts/packages/*.tgz)
	@npm ci --silent
	@touch node_modules/.install

install: node_modules/.install ## Install the dependencies of both halves
	@go work sync

build: enclave ## Build the binary, stamped with the version git describes
	@go build -ldflags "$(LDFLAGS)" -o $(BUILD_PATH) ./apps/engine/cmd/station
	@printf "$(GREEN)●$(RESET) Built $(BUILD_PATH)\n"

enclave: ## On a Mac, the Secure Enclave's Swift half the darwin build links; nothing elsewhere
	@if [ "$$(uname)" = Darwin ]; then $(MAKE) --no-print-directory $(ENCLAVE_LIB); fi

$(ENCLAVE_LIB): $(ENCLAVE_SRC)
	@mkdir -p $(@D)
	@for arch in arm64 x86_64; do \
		xcrun --sdk macosx swiftc -O -parse-as-library -static -emit-library -module-name StationEnclave \
			-target $$arch-apple-macos$(MACOS_FLOOR) $(ENCLAVE_SRC) -o $(@D)/$$arch.a || exit 1; \
	done
	@lipo -create $(@D)/arm64.a $(@D)/x86_64.a -output $@
	@printf "$(GREEN)●$(RESET) Built $@\n"

build-linux: ## Build the linux/arm64 binary a VM image carries
	@CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(LINUX_PATH) ./apps/engine/cmd/station
	@printf "$(GREEN)●$(RESET) Built $(LINUX_PATH)\n"

lint: node_modules/.install ## Both halves: gofmt, golangci-lint (once, over the three modules) and deadcode over Go, typescript check over the rest
	@unformatted=$$(gofmt -l ./apps/engine ./catalog ./specs); \
		if [ -n "$$unformatted" ]; then printf "gofmt would rewrite:\n%s\n" "$$unformatted"; exit 1; fi
	@$(TURBO) run lint
	@GOOS=linux $$(go tool -n golangci-lint) run ./apps/engine/... ./catalog/... ./specs/...
	@dead=$$(go tool deadcode -test -tags $(SPEC_TAGS) ./apps/engine/...); \
		if [ -n "$$dead" ]; then printf "deadcode found unreachable functions:\n%s\n" "$$dead"; exit 1; fi
	@printf "$(GREEN)●$(RESET) Lint passed\n"

fix: node_modules/.install ## Every fixer: gofmt over Go, then typescript fix
	@status=0; \
	gofmt -w ./apps/engine ./catalog ./specs || status=1; \
	npx typescript fix || status=1; \
	exit $$status

# value: AR-02: the use-case suites a second hypervisor must pass unchanged, run a
# second time over the fake vz once every spec has passed over tart's.
VZ_SUITES    := usecases/machines usecases/engine/keep

test: node_modules/.install enclave ## The Go tests (modules and the boundary), then the CLI specs against a fresh build of station and the window's scenarios against its proof build, then the VM suites again over vz
	@$(TURBO) run test --filter='!studio-station-specs'
	@go test ./apps/engine/... ./catalog/... ./specs/...
	@go test -race -tags $(SPEC_TAGS) -run TestWorldMac ./apps/engine/wire
	@go build -tags $(SPEC_TAGS) -ldflags "$(SPEC_LDFLAGS)" -o $(TEST_PATH) ./apps/engine/cmd/station && touch $(TEST_PATH)
	@$(TURBO) run build:proof --filter=station-desktop
	@STATION_SPEC_BUILT=1 npm --workspace specs exec -- vitest --run
	@STATION_SPEC_BUILT=1 STATION_SPEC_HYPERVISOR=vz npm --workspace specs exec -- vitest --run $(VZ_SUITES)

check: lint test build-linux menubar-check ## Every gate a change passes

menubar-check: ## The menu bar's swift test, then its bundle; one line of skip where no macOS SDK resolves
	@if ! $(SDK_PROBE) >/dev/null 2>&1; then \
		if [ "$${STATION_REQUIRE_MENUBAR:-}" = 1 ]; then \
			printf "menubar: no macOS SDK, and STATION_REQUIRE_MENUBAR=1 requires one\n"; exit 1; \
		fi; \
		printf "menubar: skipped, no macOS SDK (install Xcode to build apps/menubar)\n"; exit 0; \
	fi; \
	swift apps/menubar/scripts/symbols.swift apps/menubar/sources/station-kit/token/symbols.xcassets || exit 1; \
	mkdir -p .artifacts/menubar; \
	( start=$$(date +%s); \
		STATION_SYMBOLS_DRAWN=1 STATION_MENUBAR_SCRATCH=$(CURDIR)/.artifacts/menubar/release-build apps/menubar/scripts/build.sh > .artifacts/menubar/release.log 2>&1; \
		code=$$?; echo $$(($$(date +%s) - start)) > .artifacts/menubar/release.seconds; exit $$code ) & \
	release=$$!; \
	start=$$(date +%s); \
	swift test --parallel --package-path apps/menubar --scratch-path .artifacts/menubar/build; tested=$$?; \
	tested_in=$$(($$(date +%s) - start)); \
	wait $$release; built=$$?; \
	[ $$built -eq 0 ] && tail -n 1 .artifacts/menubar/release.log || cat .artifacts/menubar/release.log; \
	printf "menubar: swift test exit %s in %s s, release build exit %s in %s s, run side by side\n" $$tested $$tested_in $$built "$$(cat .artifacts/menubar/release.seconds)"; \
	[ $$tested -eq 0 ] && [ $$built -eq 0 ]

menubar: ## Build the menu bar's bundle, signed ad hoc, into $(MENUBAR_APP)
	@apps/menubar/scripts/build.sh

window: node_modules/.install $(if $(ENGINE),,build) menubar ## Build the window's bundle for darwin-arm64 with the engine and the menu bar inside (ENGINE=<path>, else this build's), signed ad hoc, into $(WINDOW_APP)
	@mkdir -p .artifacts/window/engine "$(dir $(WINDOW_APP))" && cp "$(or $(ENGINE),$(BUILD_PATH))" .artifacts/window/engine/station
	@../scripts/app-icon-compile.sh apps/desktop/src-tauri/icons/spwn.icon Spwn $(WINDOW_ICON) $(WINDOW_ICON)/Spwn.plist apps/desktop/src-tauri/icons/sizes
	@cd apps/desktop && CARGO_TARGET_DIR=$(CURDIR)/$(WINDOW_CARGO) npx tauri build --ci --no-sign --target aarch64-apple-darwin --bundles app --config '{"version":"$(WINDOW_VERSION)"}'
	@rm -rf $(WINDOW_APP)
	@cp -R $(WINDOW_CARGO)/aarch64-apple-darwin/release/bundle/macos/Spwn.app $(WINDOW_APP)
	@for key in CFBundleShortVersionString CFBundleVersion; do \
		shown=$$(plutil -extract $$key raw $(WINDOW_APP)/Contents/Info.plist); \
		if [ "$$shown" != "$(WINDOW_VERSION)" ]; then printf "window: the bundle's %s says %s, not the %s compiled in\n" "$$key" "$$shown" "$(WINDOW_VERSION)"; exit 1; fi; \
	done
	@shown=$$("$(WINDOW_ENGINE)" --version); if [ "$${shown%% (*}" != "station $(ENGINE_VERSION)" ]; then printf "window: the bundle's engine says %s, not station %s\n" "$$shown" "$(ENGINE_VERSION)"; exit 1; fi
	@plutil -lint -s "$(WINDOW_APP)/Contents/Library/LaunchAgents/com.jterrazz.spwn.engine.plist"
	@shown=$$(plutil -extract CFBundleIdentifier raw "$(WINDOW_MENUBAR)/Contents/Info.plist"); if [ "$$shown" != "com.jterrazz.station.menubar" ] || [ ! -x "$(WINDOW_MENUBAR)/Contents/MacOS/StationMenuBar" ]; then printf "window: the bundle carries no menu bar at %s\n" "$(WINDOW_MENUBAR)"; exit 1; fi
	@APPLE_SIGNING_IDENTITY=- ../scripts/sign-macos.sh sign $(WINDOW_APP)
	@../scripts/app-icon-check.sh $(WINDOW_APP)
	@printf "$(GREEN)●$(RESET) Built $(WINDOW_APP) ($(WINDOW_VERSION), its engine $(ENGINE_VERSION) and its menu bar, signed ad hoc)\n"

catalog: ## Build the Spwn Menu Catalog, the dev-only window over every story, never bundled into the menu bar
	@swift apps/menubar/scripts/symbols.swift apps/menubar/sources/station-kit/token/symbols.xcassets
	@swift build --package-path apps/menubar --scratch-path .artifacts/menubar/build --product StationCatalog
	@printf "$(GREEN)●$(RESET) Built $(CATALOG_BIN)\n"

catalog-open: catalog ## Open the Catalog's window (STATION_DESIGN=<design folder> shows the prototype beside): the owner's gesture, never a check's
	@$(CATALOG_BIN)

catalog-export: catalog ## Draw every story headless, dark and light, into $(CATALOG_OUT); FILTER=<regex> narrows it
	@$(CATALOG_BIN) --export $(CATALOG_OUT) $(if $(FILTER),--filter '$(FILTER)')

catalog-gallery: catalog ## Draw every story and write $(CATALOG_OUT)/index.html, the gallery to browse offline or on the phone (STATION_DESIGN=<design folder> adds the prototype's captures)
	@$(CATALOG_BIN) --export $(CATALOG_OUT) --gallery $(if $(FILTER),--filter '$(FILTER)') >/dev/null
	@printf "$(GREEN)●$(RESET) Wrote $(CATALOG_OUT)/index.html\n"

e2e-machine: build build-linux ## The machine specs on a real VM of this host: KIND=linux (tier 2) or KIND=macos (tier 3); never in check
	@specs/machine/run.sh $(KIND) $(LABEL)

e2e-binaries: build-linux ## The binaries the machine tiers carry (darwin/arm64, its v0.0.0-swap build, and linux/arm64, no cgo, unlike the release's darwin), cross-built where Go is, to copy to the Mac
	@CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_PATH) ./apps/engine/cmd/station
	@CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags "-X github.com/jterrazz/studio-station/apps/engine/wire.version=v0.0.0-swap" -o $(SWAP_PATH) ./apps/engine/cmd/station
	@printf "$(GREEN)●$(RESET) Built $(BUILD_PATH), $(SWAP_PATH) and $(LINUX_PATH) for the machine tiers\n"

# hazard: macOS kills (SIGKILL) a signed binary rewritten in place once it has run, so
# the build lands as a new file renamed over the old one; a running engine keeps its own.
provision: build ## Put this build on the machine (~/.local/bin/station), then run station init
	@mkdir -p $(BIN_DIR)
	@cp $(BUILD_PATH) $(INSTALL_PATH).new
	@chmod +x $(INSTALL_PATH).new
	@mv -f $(INSTALL_PATH).new $(INSTALL_PATH)
	@rm -f $(HOME)/.studio/bin/workstation $(HOME)/.studio/bin/station
	@printf "$(GREEN)●$(RESET) Installed $(INSTALL_PATH)\n"
	@$(INSTALL_PATH) init

deprovision: ## Remove the binary from ~/.local/bin; the user data stays
	@if [ -f "$(INSTALL_PATH)" ]; then \
		rm "$(INSTALL_PATH)"; \
		printf "$(GREEN)●$(RESET) Removed $(INSTALL_PATH)\n"; \
	else \
		printf "$(DIM)$(BINARY) not found at $(INSTALL_PATH)$(RESET)\n"; \
	fi

generate: ## Regenerate the catalog's projections, and on Linux the window's recordings
	@go run -tags catalog_yaml ./apps/engine/cmd/generate
	@printf "$(GREEN)●$(RESET) Regenerated apps/engine/catalog/builtin_table.go, skills/studio-toolbelt/SKILL.md, docs/reference/footprint.md, packages/api/schemas/station.package.v1.json, packages/api/schemas/station.machinefile.v1.json, apps/engine/remote/api/calls.json, packages/api, the page fixtures, the story worlds' pages and the recorded menubar stream and the window's bundled agents\n"
	@if $(SDK_PROBE) >/dev/null 2>&1; then \
		mkdir -p .artifacts/menubar; \
		STATION_UPDATE_INDEX=1 swift test --package-path apps/menubar --scratch-path .artifacts/menubar/build --filter StoryCoverageTests > .artifacts/menubar/index.log 2>&1 \
			|| { tail -30 .artifacts/menubar/index.log; exit 1; }; \
		printf "$(GREEN)●$(RESET) Regenerated apps/menubar/stories/INDEX.md and the stories' previews\n"; \
	else \
		printf "menubar: apps/menubar/stories/INDEX.md and the previews not regenerated, no macOS SDK\n"; \
	fi
	@if [ "$$(uname)" = Linux ]; then \
		mkdir -p .artifacts; \
		python3 -B apps/desktop/scripts/record.py --if-moved > .artifacts/window-record.log 2>&1 \
			|| { tail -30 .artifacts/window-record.log; exit 1; }; \
		printf "$(GREEN)●$(RESET) apps/desktop/src/fixtures/worlds, the window's recordings over the world build of this tree: %s\n" "$$(tail -1 .artifacts/window-record.log)"; \
	else \
		printf "window: apps/desktop/src/fixtures/worlds not regenerated, the world build is recorded in a Linux VM only\n"; \
	fi

clean: ## Remove build artifacts
	@rm -rf .artifacts
	@printf "$(GREEN)●$(RESET) Cleaned\n"
