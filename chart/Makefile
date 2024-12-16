.DEFAULT_GOAL := package

WITH_V ?= 0

BASE_VERSION = $(shell (git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0") | sed 's/^v//')
CHART_VERSION ?= $(shell $(MAKE) -s version)
CHART_APP_VERSION ?= $(shell $(MAKE) -s WITH_V=1 version)

.PHONY: package
package: lint build

.PHONY: gen
gen:
	@cp -f Chart.template.yaml Chart.yaml
	@sed -i "/^version:.*/c\version: $(BASE_VERSION)" Chart.yaml || echo "version: $(BASE_VERSION)" >> Chart.yaml
	@sed -i "/^appVersion:.*/c\appVersion: \"v$(BASE_VERSION)\"" Chart.yaml || echo "appVersion: \"v$(BASE_VERSION)\"" >> Chart.yaml

.PHONY: lint
lint: gen
	@helm lint .

.PHONY: build
build: gen
	@helm package . --version $(CHART_VERSION) --app-version $(CHART_APP_VERSION)

.PHONY: docs
docs: gen
	@echo "Generating docs for version $(BASE_VERSION) (appVersion: v$(BASE_VERSION))"
	@helm-docs

.PHONY: version
version:
	@TAG=$$(git describe --tags --exact-match 2>/dev/null); \
	if [ -n "$$TAG" ]; then \
		VERSION="$$TAG"; \
	else \
		VERSION="v$(BASE_VERSION)-commit-$$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")"; \
	fi; \
	if [ "$(WITH_V)" = "1" ]; then \
		echo "$$VERSION"; \
	else \
		echo "$${VERSION#v}"; \
	fi
