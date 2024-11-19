.DEFAULT_GOAL := package

WITH_V ?= 0

CHART_VERSION ?= $(shell $(MAKE) -s version)
CHART_APP_VERSION ?= $(shell $(MAKE) -s WITH_V=1 version)

.PHONY: package
package: lint build

.PHONY: lint
lint:
	@helm lint .

.PHONY: build
build:
	@helm package . --version $(CHART_VERSION) --app-version $(CHART_APP_VERSION)

.PHONY: docs
docs:
	@VERSION=$$(git describe --tags --abbrev=0 2>/dev/null || echo "0.0.0"); \
	VERSION=$${VERSION#v}; \
	cp -f Chart.yaml Chart.yaml.bak; \
	sed -i "s/^version:.*/version: $${VERSION}/" Chart.yaml; \
	sed -i "s/^appVersion:.*/appVersion: \"$${VERSION}\"/" Chart.yaml; \
	echo "Generating docs for version $${VERSION} (appVersion: v$${VERSION})"; \
	helm-docs; \
	mv -f Chart.yaml.bak Chart.yaml

.PHONY: version
version:
	@TAG=$$(git describe --tags --exact-match 2>/dev/null); \
	if [ -n "$$TAG" ]; then \
		VERSION="$$TAG"; \
	else \
		VERSION="$$(git describe --tags --abbrev=0 2>/dev/null)-commit-$$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")"; \
	fi; \
	if [ "$(WITH_V)" = "1" ]; then \
		echo "$$VERSION"; \
	else \
		echo "$${VERSION#v}"; \
	fi
