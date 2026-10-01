# avatax.eu

Website of **AVATAX A.E. – Accountants & Tax Consultants**, Athens, in English, German and Greek.
It replaces the old Drupal 7 site.

- **Hugo** static site: sources in `site/`, generated HTML in `public/`
- **nginx** container serves `public/` and redirects all old Drupal URLs (`nginx/conf.d/`)
- **formsvc**: a small Go service that sends the contact and CV forms by e-mail (`formsvc/`)
- Everything runs with **Docker Compose**, identically on the laptop and on the Hetzner server.
  Hugo and Go are **not** installed on the host; they run in containers.

Repository: not set yet. Add your remote with `git remote add origin <url>` (branch **`master`**).

```
docker-compose.yml         services: web, formsvc, build (+ hugo, mailpit with profile "dev")
docker-compose.proxy.yml   optional: attach "web" to a Dockerised reverse proxy on the server
.env.example               settings → copy to .env (not in git)
Makefile                   short forms of the docker compose commands (optional)
site/content/              page texts: <page>.en.md / .de.md / .el.md
site/static/images/        logo, member logo (EEA)
site/assets/images/        photos (processed by Hugo)
site/i18n/                 labels of buttons, forms, footer
site/layouts/, assets/     templates, CSS, JS
nginx/conf.d/              web server config + redirect map
formsvc/                   Go form service + tests (+ Dockerfile)
tests/old-urls.txt         old URLs that must keep working
docs/DOCKER.md             ← containers, folders, all docker compose commands
docs/MANUAL.md             ← how to edit content yourself
docs/DEPLOY.md             ← how to publish on Hetzner
docs/REVIEW.md             ← texts to proofread before going live
AGENTS.md                  ← rules and context for AI agents / developers
```

**Ports** (chosen so that this stack can run next to the other stack using 8080/8025/1313):
website `http://localhost:8090`, test mailbox `http://localhost:8026`, live reload `1314`.

## Prerequisites

- Docker Engine with the **Compose plugin** (`docker compose version` must work)
- git, rsync, curl. `make` is optional: the `Makefile` only holds short names for longer
  `docker compose` commands (`make help` lists them, `docs/MANUAL.md` §0 explains how it works).
- your user in the `docker` group (`sudo usermod -aG docker $USER`, then log in again)

## Quick start (laptop)

```bash
cd /home/asig/docker_data/avatax
cp .env.example .env && chmod 600 .env       # then set TOKEN_SECRET: openssl rand -hex 32
mkdir -p public public.next logs/nginx       # create the mount folders as your user (important)

# start: site with live reload + form service + test mailbox
COMPOSE_PROFILES=dev docker compose up -d --build      # or: make dev
```

- Website: **http://localhost:8090** (reloads by itself when you save a file in `site/`)
- Test mailbox for the forms: **http://localhost:8026**

```bash
make preview                                 # production build, served on http://localhost:8090
make test                                    # Go tests, link check, old-URL redirects
COMPOSE_PROFILES=dev docker compose down     # stop everything (or: make down)
```

All commands, folders and troubleshooting: **[docs/DOCKER.md](docs/DOCKER.md)**.

## Changing and publishing

1. Edit the content: **[docs/MANUAL.md](docs/MANUAL.md)**
2. Check: `make preview && make test`
3. Commit: `git add -A && git commit -m "…"`, and push: `git push` (once a remote is set)
4. Publish on the server: `make deploy`. See **[docs/DEPLOY.md](docs/DEPLOY.md)**.
