# avatax.eu – common tasks.  Run `make` for an overview.
#
# Laptop:  make dev -> edit site/ -> make preview -> commit + push master -> make deploy-test (admin.avatax.eu)
#          -> check -> merge master into live + push -> make deploy-live (avatax.eu)
# Server:  make update / make rollback  (called by make deploy-test / deploy-live over ssh)

DEPLOY_HOST ?= sysop@sigalas.eu
LIVE_PATH   ?= /home/sysop/docker_data/go_avatax
TEST_PATH   ?= /home/sysop/docker_data/go_avatax_test
LIVE_BRANCH ?= live
TEST_BRANCH ?= master
BASE        ?= http://localhost:8090

COMPOSE := docker compose
UIDGID  := $(shell id -u):$(shell id -g)

.DEFAULT_GOAL := help
.PHONY: help dev up down logs build publish preview test test-go test-links test-redirects de-only-map deploy deploy-test deploy-live _deploy stats update rollback status

help:
	@echo "Laptop"
	@echo "  make dev            start hugo live-reload + web + formsvc + mailpit"
	@echo "                      site: $(BASE)   mail inbox: http://localhost:8026"
	@echo "  make preview        stop the dev server, build the production site and serve it on $(BASE)"
	@echo "  make test           Go tests + link check + old-URL redirect check"
	@echo "  make deploy-test    branch $(TEST_BRANCH) -> test site admin.avatax.eu ($(TEST_PATH))"
	@echo "  make deploy-live    branch $(LIVE_BRANCH) -> live site avatax.eu ($(LIVE_PATH))"
	@echo "                      both deploy the pushed state of the branch, not the working folder"
	@echo "  make stats          visitor report from the server's access log -> stats/report.html"
	@echo "  make down / logs / status"
	@echo "Server"
	@echo "  make update         build site, publish it (previous version -> public.prev), restart containers"
	@echo "  make rollback       put public.prev back online"

# ---------------------------------------------------------------- laptop ----

dev:
	@mkdir -p public logs/nginx
	COMPOSE_PROFILES=dev $(COMPOSE) up -d --build
	@echo "Site:        $(BASE)  (live reload on every change in site/)"
	@echo "Mail inbox:  http://localhost:8026"

up:
	@mkdir -p public logs/nginx
	$(COMPOSE) up -d --build

down:
	COMPOSE_PROFILES=dev $(COMPOSE) down

logs:
	$(COMPOSE) logs -f --tail=50

status:
	$(COMPOSE) ps

# Production build into public.next (hugo container).
build:
	@mkdir -p public public.next public.prev logs/nginx   # create as our user, before docker creates them as root
	$(COMPOSE) run --rm build

# Swap the new build into public/ in place (the web container bind-mounts the
# directory, so it must not be moved/replaced – only its contents).
publish:
	@test -f public.next/index.html || (echo "public.next is empty – run 'make build' first" && exit 1)
	@mkdir -p public public.prev
	rsync -rlt --delete public/ public.prev/
	rsync -rlt --delete public.next/ public/
	@echo "Published. Previous version kept in public.prev/"

preview:
	-$(COMPOSE) stop hugo 2>/dev/null
	$(MAKE) build publish
	$(COMPOSE) up -d --build web formsvc
	@echo "Production build served on $(BASE)"

test: test-go test-links test-redirects

test-go:
	docker run --rm -u $(UIDGID) -e GOCACHE=/tmp/gocache -v "$(CURDIR)/formsvc":/src:z -w /src golang:1-alpine \
		sh -c 'test -z "$$(gofmt -l .)" && go vet ./... && go test -count=1 ./...'

# Checks every internal link and file reference in public/ (no network access needed).
test-links:
	docker run --rm -v "$(CURDIR)/public":/public:ro,z lycheeverse/lychee:latest \
		--offline --no-progress --root-dir /public --exclude-path /public/en/index.html --exclude '/api/' '/public/**/*.html'

# TEMPORARY (German only): English/Greek URL -> German page map for nginx; run after URLs change.
de-only-map:
	python3 -I tests/de-only-map.py > nginx/conf.d/redirects-de-only.conf

# Every old Drupal URL must answer with a redirect (301; 302 while only German is online)
# and lead to a page that exists.
test-redirects:
	@fail=0; while read -r path; do \
		case "$$path" in ''|\#*) continue;; esac; \
		code=$$(curl -s -o /dev/null -w '%{http_code}' "$(BASE)$$path"); \
		final=$$(curl -s -L -o /dev/null -w '%{http_code} %{url_effective}' "$(BASE)$$path"); \
		if { [ "$$code" = 301 ] || [ "$$code" = 302 ]; } && [ "$${final%% *}" = 200 ]; then echo "ok   $$path -> $${final#* }"; \
		else echo "FAIL $$path ($$code, final $$final)"; fail=1; fi; \
	done < tests/old-urls.txt; exit $$fail

# Test: branch master -> admin.avatax.eu (password-protected). Live: branch live -> avatax.eu.
# Both copy the committed and pushed state of the branch (git archive), never the working folder.
deploy-test:
	@$(MAKE) --no-print-directory _deploy BRANCH=$(TEST_BRANCH) TARGET=$(TEST_PATH)

deploy-live:
	@$(MAKE) --no-print-directory _deploy BRANCH=$(LIVE_BRANCH) TARGET=$(LIVE_PATH)

deploy:
	@echo "Use 'make deploy-test' (branch $(TEST_BRANCH) -> admin.avatax.eu) or 'make deploy-live' (branch $(LIVE_BRANCH) -> avatax.eu)."; exit 1

_deploy:
	@git fetch -q origin $(BRANCH)
	@test "$$(git rev-parse $(BRANCH))" = "$$(git rev-parse origin/$(BRANCH))" || (echo "Branch $(BRANCH) differs from origin/$(BRANCH) – push it first"; exit 1)
	@rm -rf .deploy && mkdir .deploy && git archive $(BRANCH) | tar -x -C .deploy
	@echo "Deploying $(BRANCH) ($$(git rev-parse --short $(BRANCH))) to $(DEPLOY_HOST):$(TARGET)"
	ssh $(DEPLOY_HOST) 'mkdir -p $(TARGET)'
	rsync -az --delete \
		--exclude .env --exclude /public/ --exclude /public.next/ --exclude /public.prev/ \
		--exclude /logs/ --exclude /stats/ --exclude /site/resources/ \
		.deploy/ $(DEPLOY_HOST):$(TARGET)/
	@rm -rf .deploy
	ssh $(DEPLOY_HOST) 'cd $(TARGET) && make update'

# Visitor report (GoAccess) from the server's nginx access log, built on the laptop.
# Includes rotated logs. Contains IP addresses: stats/ is neither committed nor deployed.
stats:
	@mkdir -p stats
	ssh $(DEPLOY_HOST) 'cd $(LIVE_PATH)/logs/nginx && zcat -f access.log*' \
		| docker run --rm -i -u $(UIDGID) -v "$(CURDIR)/stats":/out:z allinurl/goaccess - \
			--log-format=COMBINED --ignore-crawlers -o /out/report.html
	@echo "Report: $(CURDIR)/stats/report.html"

# ---------------------------------------------------------------- server ----

update:
	@test -f .env || (echo ".env missing – copy .env.example and fill in the server values" && exit 1)
	@mkdir -p public public.next public.prev logs/nginx
	$(MAKE) build publish
	$(COMPOSE) up -d --build --remove-orphans
	$(COMPOSE) exec -T web nginx -s reload   # nginx/conf.d is bind-mounted; a running container keeps the old config
	@sleep 3
	@curl -fsS -o /dev/null $$(grep -E '^WEB_BIND=' .env | cut -d= -f2 | sed 's|^|http://|')/healthz && echo "Health check OK" || (echo "HEALTH CHECK FAILED – see 'make logs'"; exit 1)

rollback:
	@test -f public.prev/index.html || (echo "no previous version in public.prev" && exit 1)
	rsync -rlt --delete public.prev/ public/
	@echo "Rolled back to the previous build."
