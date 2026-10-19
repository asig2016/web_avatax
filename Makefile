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
.PHONY: help dev up down logs build publish preview test test-go test-links test-redirects deploy deploy-test deploy-live _deploy _geoip stats stats-ips stats-reset update rollback status

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
	@echo "  make stats-ips      IP addresses per page, without crawlers (one page: URL=/de/)"
	@echo "  make stats-reset    delete the server's access logs (incl. rotated ones), stats start from zero"
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

# Every old Drupal URL must answer with a permanent redirect (301)
# and lead to a page that exists.
test-redirects:
	@fail=0; while read -r path; do \
		case "$$path" in ''|\#*) continue;; esac; \
		code=$$(curl -s -o /dev/null -w '%{http_code}' "$(BASE)$$path"); \
		final=$$(curl -s -L -o /dev/null -w '%{http_code} %{url_effective}' "$(BASE)$$path"); \
		if [ "$$code" = 301 ] && [ "$${final%% *}" = 200 ]; then echo "ok   $$path -> $${final#* }"; \
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
# Countries come from the free DB-IP "IP to Country Lite" database (CC BY 4.0, "IP Geolocation by DB-IP"),
# downloaded to stats/ and refreshed when it is older than a month (current month, else the previous one).
GEOIP_DB := stats/dbip-country-lite.mmdb
# All access logs of the live site, current and rotated (override to test with a local log).
ACCESS_LOGS ?= ssh $(DEPLOY_HOST) 'cd $(LIVE_PATH)/logs/nginx && zcat -f access.log*'
# Own visits, left out of `make stats` and `make stats-ips` (space-separated; GoAccess also takes ranges a-b).
STATS_EXCLUDE_IPS ?= 194.219.21.115

_geoip:
	@mkdir -p stats
	@if [ -z "$$(find $(GEOIP_DB) -mtime -31 2>/dev/null)" ]; then \
		for m in $$(date +%Y-%m) $$(date -d "$$(date +%Y-%m-01) -1 month" +%Y-%m); do \
			curl -fsSL "https://download.db-ip.com/free/dbip-country-lite-$$m.mmdb.gz" | gunzip > $(GEOIP_DB).tmp \
				&& mv $(GEOIP_DB).tmp $(GEOIP_DB) && echo "Country database: DB-IP Lite $$m" && break; \
		done; rm -f $(GEOIP_DB).tmp; \
	fi
	@test -f $(GEOIP_DB) || (echo "No country database in $(GEOIP_DB) – download failed"; exit 1)

stats: _geoip
	$(ACCESS_LOGS) \
		| docker run --rm -i -u $(UIDGID) -v "$(CURDIR)/stats":/out:z allinurl/goaccess - \
			--log-format=COMBINED --ignore-crawlers $(addprefix --exclude-ip=,$(STATS_EXCLUDE_IPS)) --geoip-database=/out/$(notdir $(GEOIP_DB)) -o /out/report.html
	@echo "Report: $(CURDIR)/stats/report.html"

# IP addresses per page, with the number of requests and the country:  make stats-ips  or  make stats-ips URL=/de/
# Counts only browsers (user agent "Mozilla/…" without bot/crawler words) and GET requests with
# status 200/304; leaves out images, CSS, JS, fonts, /api/ and STATS_EXCLUDE_IPS. Query strings are dropped.
# Countries: mmdblookup (in the GoAccess image) on the same database as `make stats`, one container for all IPs.
STATS_BOTS   := bot|crawl|spider|slurp|preview|headless|lighthouse|monitor|scan|archiv|facebookexternalhit
STATS_ASSETS := \.(css|js|map|png|jpe?g|gif|svg|webp|avif|ico|woff2?|ttf|xml|txt|json|webmanifest)$$

stats-ips: _geoip
	@$(ACCESS_LOGS) \
		| awk -F'"' -v u='$(URL)' -v x='$(STATS_EXCLUDE_IPS)' 'BEGIN { n = split(x, xs, " "); for (i = 1; i <= n; i++) own[xs[i]] } { \
			split($$1, a, " "); split($$2, r, " "); split($$3, s, " "); p = r[2]; sub(/\?.*/, "", p); \
			if (a[1] in own || r[1] != "GET" || (s[1] != 200 && s[1] != 304)) next; \
			if ($$6 !~ /^Mozilla\// || tolower($$6) ~ /$(STATS_BOTS)/) next; \
			if (p ~ /$(STATS_ASSETS)/ || p ~ /^\/api\// || (u != "" && p != u)) next; \
			print p, a[1] }' \
		| LC_ALL=C sort | uniq -c | LC_ALL=C sort -k2,2 -k1,1nr > stats/ips.tmp
	@test -s stats/ips.tmp || echo "No page views found."
	@awk '{ print $$3 }' stats/ips.tmp | sort -u \
		| docker run --rm -i -v "$(CURDIR)/stats":/geo:ro,z --entrypoint sh allinurl/goaccess -c \
			'while read ip; do echo "@ $$ip"; for k in iso_code "names en"; do \
				mmdblookup -f /geo/$(notdir $(GEOIP_DB)) -i "$$ip" country $$k 2>/dev/null; done; done' > stats/geo.tmp
	@awk 'FILENAME == "stats/geo.tmp" { if ($$1 == "@") ip = $$2; \
			else if (match($$0, /"[^"]*"/)) g[ip] = g[ip] (g[ip] == "" ? "" : " ") substr($$0, RSTART + 1, RLENGTH - 2); next } \
		$$2 != last { printf "%s%s\n", (FNR > 1 ? "\n" : ""), $$2; last = $$2 } \
		{ printf "  %6d  %-15s  %s\n", $$1, $$3, (g[$$3] == "" ? "-" : g[$$3]) }' stats/geo.tmp stats/ips.tmp
	@rm -f stats/ips.tmp stats/geo.tmp

# Deletes the live site's access logs on the server (current + rotated); error.log is kept.
# Runs inside the web container, because nginx and logrotate write the files as root.
stats-reset:
	@printf 'Delete all access logs in $(LIVE_PATH) on $(DEPLOY_HOST)? [y/N] '; read ans; [ "$$ans" = y ] || (echo "Aborted."; exit 1)
	ssh $(DEPLOY_HOST) 'cd $(LIVE_PATH) && $(COMPOSE) exec -T web sh -c "rm -f /var/log/nginx/access.log.* && : > /var/log/nginx/access.log"'
	@rm -f stats/report.html
	@echo "Access logs reset. Run 'make stats' later for a fresh report."

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
