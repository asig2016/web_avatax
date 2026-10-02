# Docker & Docker Compose reference

Everything runs in containers, both on the laptop and on the Hetzner server. The only things you need on the
host are **Docker Engine with the Compose plugin** (`docker compose …`, not the old
`docker-compose`), **git**, **rsync** and **curl**. **make** is optional: every `make` target is just a short
form of the `docker compose` commands shown below.

All commands are run in the project directory:

```bash
cd /home/asig/docker_data/avatax
```

---

## 1. Containers

| Service (container) | Image | Started by | Purpose | Port on the host |
|---|---|---|---|---|
| `web` (`avatax-web`) | `nginx:stable-alpine` | always | serves `public/`, old-URL redirects, proxies `/api/` to formsvc | `127.0.0.1:8090` (`WEB_BIND` in `.env`) |
| `formsvc` (`avatax-formsvc`) | built from `./formsvc` | always | contact and CV forms → e-mail | none (internal 8081) |
| `hugo` (`avatax-hugo`) | `hugomods/hugo:exts-non-root-0.154.5` | profile **dev** (laptop) | live-reload dev server, writes into `public/` | `127.0.0.1:1314` (live-reload only) |
| `mailpit` (`avatax-mailpit`) | `axllent/mailpit` | profile **dev** (laptop) | catches all form mails | `127.0.0.1:8026` (web inbox) |
| `build` | `hugomods/hugo:…` | profile **build**, one-off `run` | production build into `public.next/` | none |

Profiles: `COMPOSE_PROFILES=dev` starts `hugo` and `mailpit` too. The `build` service never starts on `up`;
it is only used with `docker compose run --rm build`.

## 2. Folders on the host (bind mounts)

Nothing is baked into images. All content and output live next to `docker-compose.yml`:

| Host folder | Mounted into | Notes |
|---|---|---|
| `site/` | `hugo`, `build` → `/src` | **the website sources**: edit here |
| `public/` | `web` → `/usr/share/nginx/html` (read-only); `hugo` → `/out` | the live website |
| `public.next/` | `build` → `/out` | fresh production build, then copied into `public/` |
| `public.prev/` | – | previous version, for rollback |
| `nginx/conf.d/` | `web` → `/etc/nginx/conf.d` (read-only) | web server config and redirect map |
| `logs/nginx/` | `web` → `/var/log/nginx` | access and error logs |
| `.env` | `formsvc` (env_file) and Compose variables | settings and the SMTP password; **not in git** |

> **Create the folders before the first start:** `mkdir -p public public.next logs/nginx`.
> If a bind-mount folder is missing, Docker creates it **owned by root**. The Hugo container (running as your
> user, `HOST_UID`/`HOST_GID` in `.env`) can then not write to it.
> Fix: `sudo chown -R "$(id -u):$(id -g)" public public.next logs`.

`:z` on the mounts relabels them for SELinux (Fedora). On hosts without SELinux it does nothing.

## 3. Everyday commands: `make` and the plain Docker equivalent

The `Makefile` is a list of named shortcuts (*targets*). `make <target>` runs the commands listed under that
target, and only works in the folder that contains the `Makefile`. `make` or `make help` prints the list.
A target may call others (`preview` = `build` + `publish` + start `web`/`formsvc`; `test` = three tests).
The table shows what each target runs, so everything can also be done without `make`.

| Task | make | docker compose (and helpers) |
|---|---|---|
| **Start the dev stack** (laptop) | `make dev` | `mkdir -p public logs/nginx && COMPOSE_PROFILES=dev docker compose up -d --build` |
| Stop everything | `make down` | `COMPOSE_PROFILES=dev docker compose down` |
| Status | `make status` | `docker compose --profile dev ps` |
| Follow logs | `make logs` | `docker compose logs -f --tail=50` (one service: `… logs -f formsvc`) |
| **Production build** | `make build` | `mkdir -p public.next && docker compose run --rm build` |
| Put the build online | `make publish` | `rsync -a --delete public/ public.prev/ && rsync -a --delete public.next/ public/` |
| **Preview the production site** (laptop) | `make preview` | `docker compose stop hugo` → build → publish (both above) → `docker compose up -d web formsvc` |
| Rollback to the previous build | `make rollback` | `rsync -a --delete public.prev/ public/` |
| Rebuild formsvc after Go changes | – | `docker compose up -d --build formsvc` |
| Reload nginx after config changes | – | `docker compose restart web` (check first: see §5) |
| Update base images | – | `docker compose pull && docker compose up -d` (then build again) |
| **Server: build + publish + restart + health check** | `make update` | build → publish → `docker compose up -d --build --remove-orphans` → `curl -fsS http://127.0.0.1:8090/healthz` |
| **Deploy from the laptop** | `make deploy` | `rsync` the project to the server (see `docs/DEPLOY.md` §3), then run `make update` there |

Why `rsync` instead of moving folders: `web` mounts the folder `public/` itself. Replacing the folder
(`mv`) would leave nginx serving the old, moved folder. Only the **contents** may be replaced.

## 4. Tests

| Test | make | Plain command |
|---|---|---|
| Go tests of formsvc | `make test-go` | `docker run --rm -u "$(id -u):$(id -g)" -e GOCACHE=/tmp/gocache -v "$PWD/formsvc":/src:z -w /src golang:1-alpine sh -c 'go vet ./... && go test ./...'` |
| Broken links in `public/` | `make test-links` | `docker run --rm -v "$PWD/public":/public:ro,z lycheeverse/lychee:latest --offline --no-progress --root-dir /public --exclude '/api/' '/public/**/*.html'` |
| Old Drupal URLs → 301 | `make test-redirects` | needs `web` running on :8090; checks every line of `tests/old-urls.txt` |
| All | `make test` | – |

Against another host: `make test-redirects BASE=https://new.avatax.eu`.

## 5. Checks and troubleshooting

```bash
# nginx config valid? (run before "docker compose restart web")
docker compose exec web nginx -t

# health of the web container
curl -s http://127.0.0.1:8090/healthz          # -> ok
docker inspect --format '{{.State.Health.Status}}' avatax-web

# form service reachable through nginx?
curl -s http://127.0.0.1:8090/api/token        # -> {"token":"..."}

# a shell inside a container
docker compose exec web sh
```

| Symptom | Cause / fix |
|---|---|
| `permission denied` when Hugo writes | the bind-mount folder belongs to root (see §2), so run `chown` |
| `port is already allocated` on 8090 | another container or service uses it. Change `WEB_BIND` in `.env`, e.g. `127.0.0.1:8091` |
| Dev site does not update after saving | `docker compose logs hugo` usually shows a typo in the front matter (file and line) |
| `web` restarting in a loop | `docker compose logs web`, usually an nginx config error. Check with `nginx -t` (above) |
| Form: "technical problem" | `docker compose logs formsvc` shows the SMTP error |
| Old images / disk full | `docker image prune` (removes unused images only) |

## 6. Settings (`.env`)

Copy `.env.example` to `.env` (`chmod 600 .env`) and adjust:

| Variable | Laptop | Server |
|---|---|---|
| `COMPOSE_PROFILES` | `dev` | *(empty)* |
| `WEB_BIND` | `127.0.0.1:8090` | `127.0.0.1:8090` (host nginx proxies to it) |
| `HOST_UID` / `HOST_GID` | output of `id -u` / `id -g` | same on the server |
| `SMTP_*` | `mailpit`, port `1025`, `SMTP_TLS=none` | `mail.sigalas.eu`, `465`, `tls`, user/password |
| `MAIL_FROM`, `MAIL_TO_CONTACT`, `MAIL_TO_CV` | anything (mailpit) | real addresses |
| `TOKEN_SECRET` | `openssl rand -hex 32` | `openssl rand -hex 32` (a different value) |
| `COMPOSE_FILE`, `PROXY_NETWORK`, `SITE_HOSTS` | – | only if the server proxy runs in Docker (see `docs/DEPLOY.md`) |

Changes to `.env` take effect with `docker compose up -d` (Compose recreates the affected containers).
