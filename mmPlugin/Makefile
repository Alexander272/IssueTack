# Makefile для сборки dist-пакета MM-плагина IssueTrack.
# Итог: dist/issuetrack-<версия>.tar.gz — готов к загрузке в System Console.

MM_VER ?= 0.1.0
PLUGIN_ID = issuetrack

DIST = dist
TAR = $(DIST)/$(PLUGIN_ID)-$(MM_VER).tar.gz

.PHONY: all dist clean server webapp

all: dist

server:
	@echo "Building server plugin..."
	cd server && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o dist/plugin-linux-amd64 .
	@echo "Server plugin built."

webapp:
	@if [ -f webapp/package.json ]; then \
		echo "Building webapp..."; \
		cd webapp && npm run build; \
	else \
		echo "webapp/package.json not found — skipping webapp build"; \
	fi

dist: server webapp
	@echo "Packaging plugin..."
	@mkdir -p $(DIST)
	@rm -rf $(DIST)/pack
	@mkdir -p $(DIST)/pack/server/dist $(DIST)/pack/webapp/dist
	@cp plugin.json $(DIST)/pack/
	@cp server/dist/$(shell basename server/dist/plugin-linux-amd64) $(DIST)/pack/server/dist/
	@if [ -d webapp/dist ]; then cp -r webapp/dist/. $(DIST)/pack/webapp/dist/; fi
	@tar -czf $(TAR) -C $(DIST)/pack .
	@echo "Packaged $(TAR)"

clean:
	@rm -rf $(DIST) server/dist webapp/dist
	@echo "Cleaned."