# avatax.eu – common tasks.  Run `make` for an overview.
#
# Laptop:  make dev  ->  edit site/  ->  make preview  ->  make deploy
# Server:  make update / make rollback  (called by `make deploy` over ssh)

DEPLOY_HOST ?= sysop@sigalas.eu
DEPLOY_PATH ?= /home/sysop/docker_data/go_avatax
BASE        ?= http://localhost:8090

COMPOSE := docker compose
UIDGID  := $(shell id -u):$(shell id -g)

.DEFAULT_GOAL := help
.PHONY: help dev up down logs build publish preview test test-go test-links test-redirects deploy update rollback status

help:
	@echo "Laptop"
	@echo "  make dev            start hugo live-reload + web + formsvc + mailpit"
	@echo "                      site: $(BASE)   mail inbox: http://localhost:8026"
	@echo "  make preview        stop the dev server, build the production site and serve it on $(BASE)"
	@echo "  make test           Go tests + link check + old-URL redirect check"
	@echo "  make deploy         copy the project to $(DEPLOY_HOST):$(DEPLOY_PATH) and run 'make update' there"
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
	@mkdir -p public.next
	$(COMPOSE) run --rm build

# Swap the new build into public/ in place (the web container bind-mounts the
# directory, so it must not be moved/replaced – only its contents).
publish:
	@test -f public.next/index.html || (echo "public.next is empty – run 'make build' first" && exit 1)
	@mkdir -p public public.prev
	rsync -a --delete public/ public.prev/
	rsync -a --delete public.next/ public/
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

# Every old Drupal URL must answer 301 and lead to a page that exists.
test-redirects:
	@fail=0; while read -r path; do \
		case "$$path" in ''|\#*) continue;; esac; \
		code=$$(curl -s -o /dev/null -w '%{http_code}' "$(BASE)$$path"); \
		final=$$(curl -s -L -o /dev/null -w '%{http_code} %{url_effective}' "$(BASE)$$path"); \
		if [ "$$code" = 301 ] && [ "$${final%% *}" = 200 ]; then echo "ok   $$path -> $${final#* }"; \
		else echo "FAIL $$path ($$code, final $$final)"; fail=1; fi; \
	done < tests/old-urls.txt; exit $$fail

deploy:
	@test -z "$$(git status --porcelain)" || (echo "Uncommitted changes – commit first (git add -A && git commit)"; exit 1)
	ssh $(DEPLOY_HOST) 'mkdir -p $(DEPLOY_PATH)'
	rsync -az --delete \
		--exclude .git --exclude .env --exclude /public/ --exclude /public.next/ --exclude /public.prev/ \
		--exclude /logs/ --exclude /site/resources/ \
		./ $(DEPLOY_HOST):$(DEPLOY_PATH)/
	ssh $(DEPLOY_HOST) 'cd $(DEPLOY_PATH) && make update'

# ---------------------------------------------------------------- server ----

update:
	@test -f .env || (echo ".env missing – copy .env.example and fill in the server values" && exit 1)
	@mkdir -p logs/nginx
	$(MAKE) build publish
	$(COMPOSE) up -d --build --remove-orphans
	@sleep 3
	@curl -fsS -o /dev/null $$(grep -E '^WEB_BIND=' .env | cut -d= -f2 | sed 's|^|http://|')/healthz && echo "Health check OK" || (echo "HEALTH CHECK FAILED – see 'make logs'"; exit 1)

rollback:
	@test -f public.prev/index.html || (echo "no previous version in public.prev" && exit 1)
	rsync -a --delete public.prev/ public/
	@echo "Rolled back to the previous build."
