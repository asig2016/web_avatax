# Publishing avatax.eu on the Hetzner server

This is a step-by-step runbook. Work through it top to bottom the first time. For later
updates, only **section 6 (Updating)** applies.

Commands are given as `make` targets with the plain `docker compose` equivalent where it matters.
The full command reference (containers, folders, troubleshooting) is in **[DOCKER.md](DOCKER.md)**.

| | |
|---|---|
| Server | `135.181.76.115` (Hetzner), which also hosts other sites (incl. another Docker stack of this kind on port 8080) behind an existing nginx reverse proxy |
| SSH login | `ssh sysop@sigalas.eu` (same server as sigalas.eu; used by `make deploy`, see `DEPLOY_HOST` in the Makefile) |
| Project directory | laptop `/home/asig/docker_data/avatax`, server `/home/sysop/docker_data/go_avatax` |
| Containers | `avatax-web` (nginx, static site), `avatax-formsvc` (forms → e-mail); on the server **no** `hugo`/`mailpit` |
| Source code | GitHub `asig2016/web_avatax`, branch `master` |
| DNS | papaki.gr (`dns1/dns2.papaki.gr`) |
| Mail relay | `mail.sigalas.eu` (194.219.21.115, a different machine) – the **current MX of avatax.eu** (shared mail server). Use whatever mail server avatax.eu mail is handled by. |

**Port:** this stack listens on `127.0.0.1:8090`, so it can run next to the other stack on the same server
(which uses 8080). Containers are named `avatax-*`, the Compose project is `avatax`.

The new stack **does not open any public port** and does not handle TLS. The existing
proxy keeps doing that, and the `web` container only listens on `127.0.0.1:8090` or on the proxy's
Docker network.

---

## 0. Prerequisites on the laptop

```bash
cd /home/asig/docker_data/avatax
make preview && make test     # production build + Go tests, link check, redirect check
git status                    # everything committed? (make deploy refuses uncommitted changes)
git push                      # GitHub copy up to date
```

`make deploy` connects as `sysop@sigalas.eu` (set in the Makefile as `DEPLOY_HOST`; override with
`make deploy DEPLOY_HOST=…` if needed). Make sure key login works without a password prompt:

```
ssh-copy-id sysop@sigalas.eu      # once, if not done yet
```

Then test it with `ssh sysop@sigalas.eu 'docker --version && docker compose version && make --version | head -1 && rsync --version | head -1'`.
The server needs **Docker Engine with the Compose plugin** (`docker compose`, not the old `docker-compose`),
**make, rsync and curl**. For example, on Debian/Ubuntu: `sudo apt install make rsync curl`.
Hugo and Go are **not** needed on the server; they run in containers.

The server user must be able to run `docker` without sudo (`sudo usermod -aG docker sysop`, then log in again).

---

## 1. Inspect the server (determine variant A or B)

```bash
ssh sysop@sigalas.eu
docker ps --format 'table {{.Names}}\t{{.Image}}\t{{.Ports}}'
docker network ls
sudo ss -tlnp | grep -E ':80 |:443 '
```

- **Variant A – the proxy runs in Docker.** `docker ps` shows a container such as
  `nginxproxy/nginx-proxy` (usually with `nginxproxy/acme-companion`), or
  `jc21/nginx-proxy-manager`, publishing ports 80/443. Note the name of **its Docker network**
  (`docker inspect <proxy-container> -f '{{json .NetworkSettings.Networks}}'`).
- **Variant B – nginx runs directly on the host.** `ss` shows an `nginx` process (not
  `docker-proxy`) on 80/443. The configuration is in `/etc/nginx/`.

Also find out where the old Drupal site is served from, so you can restore it if needed:

```bash
# Variant B:
grep -rl 'avatax' /etc/nginx/ 2>/dev/null
# Variant A: look for a container / proxy host for avatax.eu
docker ps | grep -i -E 'drupal|php|avatax'
```

## 2. Back up the old Drupal site (do not skip)

Adapt the paths to what you found in step 1:

```bash
ssh sysop@sigalas.eu
mkdir -p ~/backup-drupal-$(date +%F) && cd ~/backup-drupal-$(date +%F)
sudo tar czf drupal-files.tgz -C /var/www avatax.eu          # docroot (adjust)
mysqldump -u root -p <drupal_db> | gzip > drupal-db.sql.gz     # or: docker exec <db> mysqldump ...
sudo cp -a /etc/nginx/sites-enabled/avatax* . 2>/dev/null     # variant B: the old vhost
exit
rsync -av avatax:backup-drupal-*/ ~/backup/avatax-drupal/    # copy to the laptop
```

## 3. First installation on the server

```bash
# on the laptop – copies everything except .env, public/, logs/, .git
cd /home/asig/docker_data/avatax
ssh sysop@sigalas.eu 'mkdir -p /home/sysop/docker_data/go_avatax'   # in the home directory of sysop – no sudo needed
rsync -az --exclude .git --exclude .env --exclude /public/ --exclude /public.next/ \
      --exclude /public.prev/ --exclude /logs/ --exclude /site/resources/ \
      ./ sysop@sigalas.eu:/home/sysop/docker_data/go_avatax/
```

Create the server's `.env`:

```bash
ssh sysop@sigalas.eu
cd /home/sysop/docker_data/go_avatax
cp .env.example .env && chmod 600 .env
nano .env
```

Server values:

```ini
COMPOSE_PROFILES=
WEB_BIND=127.0.0.1:8090
HOST_UID=1000            # output of `id -u` on the server
HOST_GID=1000            # output of `id -g`

SMTP_HOST=mail.sigalas.eu
SMTP_PORT=587
SMTP_TLS=starttls
SMTP_USER=website@avatax.eu      # a mailbox that may send via mail.sigalas.eu
SMTP_PASS=********
MAIL_FROM=Website avatax.eu <website@avatax.eu>
MAIL_TO_CONTACT=<address that receives enquiries>
MAIL_TO_CV=<address that receives applications, optional>
TOKEN_SECRET=<output of: openssl rand -hex 32>

# Variant A only:
# COMPOSE_FILE=docker-compose.yml:docker-compose.proxy.yml
# PROXY_NETWORK=<network name from step 1>
# SITE_HOSTS=new.avatax.eu        # staging first, see step 4
```

Check that the server can reach the mail relay:
`nc -vz mail.sigalas.eu 587`. If it fails, the mail server's firewall must allow `135.181.76.115`.

Create the mount folders as your user (otherwise Docker creates them owned by root), then build and start:

```bash
mkdir -p public public.next public.prev logs/nginx
make update          # builds the site, starts web + formsvc, runs a health check
```

Without `make`, the same steps are:

```bash
docker compose run --rm build                                   # Hugo → public.next/
rsync -a --delete public/ public.prev/ && rsync -a --delete public.next/ public/
docker compose up -d --build --remove-orphans                   # web + formsvc
docker compose ps                                               # both "running", web "healthy"
curl -s http://127.0.0.1:8090/healthz                           # -> ok
curl -s http://127.0.0.1:8090/api/token                         # -> {"token":"..."}
```

The old site is still online at this point, because nothing points to the new containers yet.

## 4. Staging: test under `new.avatax.eu` first

1. **DNS at papaki.gr:** add an `A` record `new` → `135.181.76.115`. Wait until
   `dig +short new.avatax.eu` returns the IP.
2. Point the proxy at the new stack:

   **Variant A – nginx-proxy + acme-companion:** with `SITE_HOSTS=new.avatax.eu` in `.env`, run
   `docker compose up -d`. The proxy picks up `VIRTUAL_HOST`/`LETSENCRYPT_HOST` automatically and
   issues a certificate. **Allow CV uploads:** nginx-proxy limits request bodies to 1 MB by default. Create
   `<proxy vhost.d volume>/new.avatax.eu` (later also `avatax.eu`) containing
   `client_max_body_size 6m;` and restart the proxy.

   **Variant A – Nginx Proxy Manager:** go to *Hosts → Proxy Hosts → Add*. Set the domain to `new.avatax.eu`,
   scheme `http`, forward host `avatax-web`, port `80`, and enable *Block common exploits*. Under *SSL*,
   request a new Let's Encrypt certificate and enable *Force SSL*. Under *Advanced*, add
   `client_max_body_size 6m;`. The NPM container must be on the network named in `PROXY_NETWORK`.

   **Variant B – host nginx:** copy `docs/host-nginx-avatax.conf` to the nginx config
   directory, replace `avatax.eu` with `new.avatax.eu` in `server_name` and the
   certificate paths (for staging, drop the `www` server block), then run:
   ```bash
   sudo nginx -t && sudo systemctl reload nginx
   sudo certbot --nginx -d new.avatax.eu
   ```
3. Test everything on https://new.avatax.eu:
   - click through all pages in EN / DE / EL, including the language switch;
   - send **one contact message and one application with a PDF per language** and check that they arrive
     in the recipient mailbox (look in the spam folder as well; see "Mail deliverability" below);
   - test old URLs, for example `curl -sI https://new.avatax.eu/de/uber-uns` → `301` → `/de/ueber-uns/`,
     or run the whole list from the laptop:
     `make test-redirects BASE=https://new.avatax.eu`.

## 5. Cut-over: avatax.eu → new site

1. Keep the old Drupal proxy entry or vhost (do not delete it), so you can switch back if needed:
   - **Variant B:** `sudo mv /etc/nginx/sites-enabled/<old-avatax-vhost> /root/avatax-drupal.conf.bak`
     (on RHEL-like systems: `/etc/nginx/conf.d/…`).
   - **Variant A:** remove `VIRTUAL_HOST` from the old Drupal container, or disable its NPM proxy host.
2. Switch to the new stack:
   - **Variant B:** install `docs/host-nginx-avatax.conf` unchanged (avatax.eu + www), then run:
     ```bash
     sudo nginx -t && sudo systemctl reload nginx
     sudo certbot --nginx -d avatax.eu -d www.avatax.eu     # certificate for avatax.eu + www
     ```
   - **Variant A:** set `SITE_HOSTS=avatax.eu,www.avatax.eu` in `.env` and run `docker compose up -d`
     (nginx-proxy), or add both domains to the NPM proxy host.
3. **Certificate:** on 2026-10-01 the server answered `https://avatax.eu` with a certificate issued for **avatax.gr**
   (so browsers may warn for avatax.eu). Make sure the new certificate covers `avatax.eu` and `www.avatax.eu`
   (and avatax.gr, if that domain should keep working) and that auto-renewal works:
   - Variant B: `systemctl list-timers | grep certbot` must list a timer, and
     `sudo certbot renew --dry-run` must succeed.
   - Variant A: `docker logs <acme-companion>` or the NPM *SSL Certificates* page must show a valid
     certificate for avatax.eu.
4. Post-checks:
   ```bash
   curl -sI https://avatax.eu/ | head -1                           # 200
   curl -sI http://avatax.eu/ | grep -i location                   # -> https://avatax.eu/
   curl -sI https://www.avatax.eu/ | grep -i location              # -> https://avatax.eu/
   curl -sI https://avatax.eu/de/uber-uns | grep -i location       # -> /de/ueber-uns/
   curl -sI https://avatax.eu/images/logo.png | head -1                # 200
   make test-redirects BASE=https://avatax.eu                     # on the laptop
   ```
   Send one more real message through the contact form.
5. In **Google Search Console**, submit `https://avatax.eu/sitemap.xml`.
6. After the cut-over you can remove the DNS record `new.avatax.eu`.

## 6. Updating the site (normal workflow)

```bash
# on the laptop
cd /home/asig/docker_data/avatax
make dev                      # http://localhost:8090 with live reload, mails -> http://localhost:8026
#   … edit files in site/content/, site/data/, site/static/files/ …
make preview && make test     # production build + checks
git add -A && git commit -m "Update …"
git push                      # GitHub (backup)
make deploy                   # rsync to the server + `make update` there
```

`make deploy` copies the project with `rsync` (without `.git`, `.env`, `public*/`, `logs/`) and then runs
`make update` on the server: `docker compose run --rm build` into `public.next/`, live version →
`public.prev/`, new build → `public/`, `docker compose up -d --build` (rebuilds formsvc only if its
code changed) and a health check.

**Rollback** (if a deploy went wrong): `ssh sysop@sigalas.eu 'cd /home/sysop/docker_data/go_avatax && make rollback'`
(plain: `rsync -a --delete public.prev/ public/`). This restores the previous build immediately; nginx
does not need a restart.

**Emergency switch back to Drupal:** restore the saved proxy entry or vhost from step 5.1 and reload nginx.

### Quick edits directly on the server

All content is on the server's file system, so you can edit it there directly:

```bash
ssh sysop@sigalas.eu
cd /home/sysop/docker_data/go_avatax
nano site/content/contact.el.md
make build publish            # live within seconds (plain: see DOCKER.md §3)
```

**Important:** the next `make deploy` from the laptop overwrites server-side edits. Make the
same change on the laptop too, or copy it back first:
`rsync -av sysop@sigalas.eu:/home/sysop/docker_data/go_avatax/site/ ./site/`.

## 7. Common content tasks

Details for all content tasks are in **[MANUAL.md](MANUAL.md)**. The most common ones:

| Task | What to do |
|---|---|
| New job ad | Copy `site/content/careers/voithos-logisti.el.md`, then adjust `title`, `url` and `positionName`, and set `active: true`. It then appears in all three careers pages and in the CV form |
| Close a job ad | Set `active: false` in its front matter |
| Change text | Edit `site/content/<page>.<lang>.md` (`en`, `de`, `el`) |
| Address / phone / member logos | `site/hugo.toml` (`[params]` and `[languages.*.params]`) |
| Form and button labels | `site/i18n/{en,de,el}.toml` |
| Enable Google Analytics | Set `ga4ID = "G-…"` in `site/hugo.toml`. The consent banner then appears automatically |

## 8. Operations

- **Logs (server):** `/home/sysop/docker_data/go_avatax/logs/nginx/{access,error}.log`, plus
  `docker compose logs formsvc` (records every sent mail and every rejected spam attempt).
- **Status:** `docker compose ps` (`web` must be `healthy`), `curl -s http://127.0.0.1:8090/healthz`.
- **Log rotation:** the privacy policy promises to keep logs for at most 30 days. Create
  `/etc/logrotate.d/avatax`:
  ```
  /home/sysop/docker_data/go_avatax/logs/nginx/*.log {
      daily
      rotate 30
      compress
      missingok
      notifempty
      copytruncate
  }
  ```
- **Updates of the base images** (nginx, Go, distroless), about monthly:
  `docker compose pull && make update` (`pull` refreshes nginx; `update` rebuilds formsvc with the
  newest Go image). The Hugo version is pinned in `docker-compose.yml` (`HUGO_IMAGE_TAG`). Change it
  deliberately and test locally first.
- **Disk space:** `docker image prune` removes unused old images.
- **Containers start automatically** after a reboot (`restart: unless-stopped`), provided the Docker
  service is enabled (`systemctl is-enabled docker`).
- **Backup:** everything except `.env` is in git on the laptop **and on GitHub**
  (GitHub `asig2016/web_avatax`, branch `master`). Back up the server's `.env` separately; it
  contains the SMTP password. The generated `public/` needs no backup; it can be rebuilt at any time.

### Mail deliverability

The form mails are sent **from** `MAIL_FROM` through `mail.sigalas.eu` with SMTP authentication,
so SPF/DKIM are those of your own mail server. The visitor's address is only set as **Reply-To**,
which means "Reply" in the mail client answers the visitor directly. If mails end up in spam, check that
the SPF record of avatax.eu allows mail.sigalas.eu and that `SMTP_USER` may send as `MAIL_FROM`.

### Troubleshooting

| Symptom | Check |
|---|---|
| Form says "technical problem" | `docker compose logs formsvc` shows the SMTP error (authentication, TLS, firewall) |
| Form says "could not be verified" | The page was open for more than 3 hours, or it was submitted within 3 seconds. Reload the page |
| CV upload fails with larger files | The proxy's `client_max_body_size` (see step 4) |
| 502 Bad Gateway | `docker compose ps`. Is `avatax-web` running and healthy? Variant A: is it on the proxy network (`docker network inspect <proxy-network>`)? |
| `web` keeps restarting | `docker compose logs web`, usually an nginx config error. Check with `docker compose exec web nginx -t` |
| Permission denied in `public*/` or `logs/` | the folders were created by Docker as root: `sudo chown -R "$(id -u):$(id -g)" public* logs` |
| Changes not visible | Did you run `make update` or `make build publish`? Reload the browser without cache |
