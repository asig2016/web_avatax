# Manual: editing the website content yourself

This manual shows how to change texts, add pages and publish job ads.
You need no programming knowledge, only a text editor (e.g. VS Code, gedit or `nano`).

---

## 0. Terminal and `make`: how the commands work

All commands in this manual are typed in a **terminal**, always in the project folder:

- In Dolphin, open the folder `/home/asig/docker_data/avatax` and press **F4** (terminal panel),
  or right-click → *Open Terminal Here*.
- Or open any terminal and type `cd /home/asig/docker_data/avatax`.

### What is the `Makefile`?

The file **`Makefile`** in the project folder is a list of named **shortcuts** (called *targets*). Each shortcut
bundles one or more longer commands under a short name. The program `make` reads the `Makefile` in the
**current folder** and runs the commands of the shortcut you name. Example from the file:

```make
dev:
	@mkdir -p public logs/nginx
	COMPOSE_PROFILES=dev docker compose up -d --build
```

Typing `make dev` runs exactly these two lines: it creates the folders `public` and `logs/nginx` if needed,
then starts the Docker containers. (The `@` only hides that line from the screen output.)

| You type | What happens |
|---|---|
| `make` or `make help` | shows the list of all shortcuts |
| `make dev` | starts the test site with live reload → http://localhost:8090, test mails → http://localhost:8026 |
| `make preview` | builds the final website and shows it on http://localhost:8090 |
| `make test` | automatic checks: broken links, old addresses, form service |
| `make down` | stops all containers |
| `make deploy` | publishes the site on the server |

Good to know:

- **Only in the project folder:** elsewhere `make` reports *"No targets specified and no makefile found"*.
- **Shortcuts can call other shortcuts:** `make preview` runs `build` and `publish`; `make test` runs three tests.
- **Optional:** every shortcut and its plain `docker compose` command is listed in `docs/DOCKER.md`.
- **Don't double-click the `Makefile`** in Dolphin. It is a text file you can read in an editor, but it only
  "runs" through the `make` command in a terminal.

---

## 1. Start the test site on your laptop

The test site runs in Docker containers. You only need Docker; everything else is inside the containers.

```bash
cd /home/asig/docker_data/avatax
make dev
# without make:  COMPOSE_PROFILES=dev docker compose up -d --build
```

Open **http://localhost:8090** in the browser. Every time you save a file, the page reloads by itself
after 1–2 seconds. Messages sent through the forms don't go out as real e-mails; you can read them at
**http://localhost:8026**.

To stop the test site: `make down` (without make: `COMPOSE_PROFILES=dev docker compose down`).

First time on a new computer: see the *Quick start* in `README.md` (copy `.env`, create folders).

---

## 2. Where is what?

All texts are in **`site/content/`**. Each page exists three times, once per language:

```
site/content/
├── _index.en.md / .de.md / .el.md          Home page
├── about.*.md                              About us / Über uns / Η εταιρεία
├── contact.*.md                            Contact page (text next to the form)
├── privacy.*.md                            Privacy policy
├── services/
│   ├── _index.*.md                         Services overview
│   ├── accounting.*.md                     Accounting & financial statements
│   ├── payroll.*.md                        Payroll
│   ├── tax.*.md                            Tax consultancy
│   ├── fiscal-representation.*.md          Fiscal representation
│   ├── accounting-department-support.*.md  Organisation & support of accounting departments
│   └── start-in-greece.*.md                Your start in Greece
└── careers/
    ├── _index.*.md                         Careers page
    ├── apply.*.md                          Application form page
    └── <job>.el.md                         Job ads (Greek only)
```

`.en.md` = English, `.de.md` = German, `.el.md` = Greek.

---

## 3. Structure of a page file

Every file has two parts: the **settings block** between the two `---` lines, and the **text** below it.

```markdown
---
title: "Tax consultancy in Greece"          ← large heading on the page (H1)
linkTitle: "Tax consultancy"                ← short name in the menu
seoTitle: "Tax Advisor in Greece – Tax Consultancy in Athens"   ← title in Google (max. 60 characters)
description: "Tax consultancy in Greece: …"  ← text below the title in Google (max. 155 characters)
url: /services/tax-consultancy/             ← address of the page – do NOT change without a redirect
menus:
  main:
    parent: services
    weight: 20                              ← position in the menu (smaller = further up)
---
Here comes the text of the page.

## A sub-heading

- a list item
- another list item
```

**Important rules**

- Keep the `"` quotation marks around values in the settings block.
- Indentation in the settings block uses **spaces**, never tabs.
- **Never change `url:`** of an existing page unless you also add a redirect (see section 9).

---

## 4. Formatting the text (Markdown)

| You write | Result |
|---|---|
| `## Heading` | sub-heading |
| `### Smaller heading` | smaller sub-heading |
| `**bold**` | **bold** |
| `*italic*` | *italic* |
| `- item` | bullet list |
| `1. item` | numbered list |
| `[link text](/de/kontakt/)` | link to a page of the site |
| `[link text](https://www.soel.gr)` | link to another website |
| empty line | new paragraph |

Tip: link your pages to each other, e.g. from Tax consultancy to Tax due diligence. It helps
visitors and Google.

---

## 5. Changing an existing text

1. Open the file, e.g. `site/content/services/tax.de.md`.
2. Change the text and save.
3. Check it on http://localhost:8090/de/leistungen/steuerberatung/.
4. Check whether the other two languages need the change too (`tax.en.md`, `tax.el.md`). The facts must stay the
   same, but the pages need **not** be word-for-word translations: Greek pages address Greek clients
   (professional terms, legal references), German and English pages address people and companies coming
   to Greece for the first time (explain what is different, compare with Germany where useful).

If a file contains a line `review: "…"`, the text is new or translated and has not yet been checked.
After you have checked it, **delete this line**.

---

## 6. Adding a new service

1. Copy an existing service in all three languages, e.g.:
   ```bash
   cd site/content/services
   cp tax.en.md new-service.en.md
   cp tax.de.md new-service.de.md
   cp tax.el.md new-service.el.md
   ```
2. In each copy, change **title, linkTitle, seoTitle, description, url, teaser, weight** and the text.
   - `url` must be unique, e.g. `/services/new-service/`, `/de/leistungen/neue-leistung/`,
     `/el/ypiresies/nea-ypiresia/`
   - `teaser`: one sentence for the card on the home page
   - `icon`: one of `book`, `person`, `tax`, `globe`, `group`, `start`, `report`, `finance`, `search`, `shield`
3. The service appears automatically in the menu, on the home page and in the services overview.

> ⚠️ AVATAX is an accounting and tax firm: do **not** describe services as statutory audits ("Wirtschaftsprüfung",
> "έλεγχος οικονομικών καταστάσεων"). Use "tax audit support" / "Betriebsprüfung" / "φορολογικός έλεγχος" instead.

### Optional: frequently asked questions on a page

Add to the settings block (Google can then show them in search results):

```yaml
faq:
  - q: "The question?"
    a: "The answer."
  - q: "Second question?"
    a: "Second answer."
```

---

## 7. Job ads

**New job ad**

1. Copy the existing ad:
   ```bash
   cp site/content/careers/voithos-logisti.el.md \
      site/content/careers/nea-thesi.el.md
   ```
2. Change `title`, `url` (e.g. `/el/karriera/nea-thesi/`), `positionName` (appears in the
   application form), `date`, `summary`, `seoTitle`, `description` and the text.
3. Set `active: true`.

The ad appears on the careers page in all three languages (in EN/DE marked "in Greek")
and can be selected in the application form.

**Close a job ad:** set `active: false`. It then disappears from the careers page and from the form.

**Careers on the German/English site (currently hidden, owner decision 2026-10-01):** the files
`content/careers/_index.{de,en}.md` and `apply.{de,en}.md` contain `build: { render: never, list: never }`, so there is
no "Karriere"/"Careers" menu entry and no page in DE/EN; `/careers/` and `/de/karriere/` redirect (301) to the Greek
careers page (`nginx/conf.d/redirects.conf`). To show them again: remove the `build:` block, add back
`menus: { main: { weight: 30 } }`, delete the six "careers hidden" lines in `redirects.conf`, then `make preview`.

---

## 8. Other changes

| What | Where |
|---|---|
| Address, phone, register no. | `site/hugo.toml` → `[params]` and `[languages.xx.params]` |
| Company name / tagline per language | `site/hugo.toml` → `[languages.xx]` |
| Member logos in the footer | `site/hugo.toml` → `[[params.members]]`, logos in `site/static/images/members/` |
| Labels of buttons, forms, cookie banner | `site/i18n/en.toml`, `de.toml`, `el.toml` |
| Home page photos | `site/assets/images/` (acropolis.jpg = top image, syngrou.jpg, office.jpg) |
| Google Analytics | `site/hugo.toml` → `ga4ID = "G-XXXXXXX"` (the cookie banner then appears automatically) |
| Address of a page changed | add a line to `nginx/conf.d/redirects.conf`: `/old/address   /new/address/;` |
| Legal notice (company data, liability note, copyright) | `site/content/legal.en.md`, `legal.de.md`, `legal.el.md` |
| Live chat on/off | `site/hugo.toml` → `rocketchatURL` and `.env` → `CHAT_URL`, `CHAT_DEPARTMENT` (empty = no chat button) |

### Live chat (Rocket.Chat)

The chat button at the bottom right appears **only while an agent of the department is *Available***. formsvc
asks Rocket.Chat (`CHAT_URL`, `CHAT_DEPARTMENT` in `.env`) at most every 30 seconds and the page asks formsvc
(`/api/chat-status`) on load and every minute, so a status change shows up within about 1.5 minutes. The
Livechat widget itself is loaded from `rocketchatURL` (same server as `CHAT_URL`) only after the visitor clicks
"Start chat".

**Settings in `.env`** (on the laptop and on the server, `/home/sysop/docker_data/go_avatax/.env`):

```bash
# ---- live chat (Rocket.Chat) ----
# Base URL of the Rocket.Chat server, without a trailing slash and without /home or /livechat.
# Must be the same server as rocketchatURL in site/hugo.toml.
CHAT_URL=https://egroupware.sigalas.eu
# Omnichannel department of this website: its name (as in Omnichannel → Departments)
# or its ID (last part of the address when editing the department).
CHAT_DEPARTMENT=AVATAX
```

To switch the chat off, leave `CHAT_URL` empty (`CHAT_URL=`). The button then never appears.
After changing `.env`, recreate the container. A plain restart does **not** read `.env` again:

```bash
docker compose up -d --force-recreate formsvc
```

Check after about 30 seconds: `curl -s http://localhost:8090/api/chat-status` on the laptop, or
`curl -s -u developer https://admin.avatax.eu/api/chat-status` for staging. While you are *Available* in Rocket.Chat
the answer is `{"online":true,"department":"…"}`. If it stays `{"online":false}`:
`docker compose logs formsvc | grep -i chat`. No `chat status:` lines means `CHAT_URL` is not set in the
container; a line such as `chat status: department "AVATAX" not found` or a timeout says what is wrong.

One-time settings in Rocket.Chat (as administrator):

1. **Iframe permission (required):** the chat window is the page `/livechat` of the chat server, shown in an iframe
   on avatax.eu. Rocket.Chat sends `X-Frame-Options: sameorigin` on all its pages (anti-clickjacking), so browsers
   leave that iframe empty on other sites. Remove the header **only for `/livechat`** in the proxy nginx in front of
   EGroupware/Rocket.Chat (server `ubuntu5`, `~/docker_data/proxy/conf.d/egroupware.conf`, inside the `server` block
   for `egroupware.sigalas.eu`, above `location /`). The block copies the proxy settings of `location /` and only
   changes the frame headers; replace `SIGALAS-SITE` with the other website's address(es):

   ```nginx
   # Rocket.Chat Livechat widget: may be embedded in our own websites only
   location /livechat {
       access_log off;

       proxy_pass http://egw_server1_backend;
       proxy_set_header X-Real-IP $remote_addr;
       proxy_set_header Host $host;
       proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
       proxy_set_header X-Forwarded-Proto "https";
       proxy_http_version 1.1;
       proxy_set_header Upgrade $http_upgrade;
       proxy_set_header Connection "Upgrade";

       proxy_hide_header X-Frame-Options;
       add_header Content-Security-Policy "frame-ancestors https://avatax.eu https://www.avatax.eu https://new.avatax.eu http://localhost:8090 https://SIGALAS-SITE" always;
   }
   ```

   The widget's websocket does not start with `/livechat` and keeps going through `location /`. The client-certificate
   rules of the other locations are not affected; `/livechat` stays public as before.
   Then `nginx -t && nginx -s reload` (in the proxy container or on the host). Check: `curl -sI https://egroupware.sigalas.eu/livechat` shows the
   `frame-ancestors` line and no `X-Frame-Options`; `curl -sI https://egroupware.sigalas.eu/home` still shows
   `X-Frame-Options: sameorigin`. Do **not** switch off *Restrict access inside any Iframe* in Rocket.Chat
   (Settings → General): that removes the protection from the whole Rocket.Chat interface, including the admin pages.
2. **Omnichannel → Livechat:** *Display offline form* = off. *Livechat allowed domains*:
   `avatax.eu, www.avatax.eu, localhost:8090` (plus the domain of the other site using this server).
3. **Omnichannel → Departments:** create **AVATAX** (name must match `CHAT_DEPARTMENT`), add the agents,
   and switch **off** *Show on registration page* – otherwise visitors could see other departments in a dropdown.
4. **Omnichannel → Custom Fields:** create `website` and `language` with scope **Room** (not *Visitor*: since
   Rocket.Chat 7 visitor fields are not shown in the contact profile), visibility *Visible*, *Public* off (otherwise the
   visitor is asked for them in the chat form). The website fills them in automatically; the agent sees them in the
   chat's **Room Information** (ⓘ icon), e.g. `website: avatax.eu`, `language: de`. Chats from staging or the laptop
   show `admin.avatax.eu` or `localhost:8090`. The *Channel* column in the Contact Center always shows the chat server,
   not the website.
5. Keep welcome, offline and trigger messages neutral (no company name), because they are shared by all sites on the
   server. Set the server-wide widget title (*Livechat Appearance* → title) to something neutral such as "Chat": it is
   shown on the chat form before the website's own title "AVATAX A.E." takes over. Switch off showing the **agent's
   e-mail address** to visitors (Settings → Omnichannel → Livechat, e.g. *Show agent email*); otherwise the visitor sees
   the agent's mailbox address in the chat header.
6. To answer chats: set your status to *Available* (Omnichannel toggle) in Rocket.Chat.
7. **Omnichannel → Custom Fields:** also create `topic` (scope **Room**, visibility *Visible*, *Public* off) for the
   chat buttons on service pages (next section).
8. **See the page directly in the chat:** Administration → Settings → Omnichannel → *Livechat* → switch on
   **Send Visitor Navigation History as a Message**. The page the visitor is on (title and address, e.g.
   "Steuervorteile beim Zuzug … | AVATAX") then appears as a system message in the conversation itself.

#### Greeting and pre-sales note

The greeting is part of the website's own chat panel (shown after a click on "Chat", before the chat server is
contacted). It says that answers in the chat are a first, non-binding assessment and that full advice requires a
written engagement, consistent with the legal notice page. The texts are in `site/i18n/*.toml`: `chatGreeting`
(greeting) and `chatNotice` (privacy note).

Flow for the visitor: **Chat → name and e-mail (in our panel, in the page language) → Start chat → message.**
The website hands name and e-mail to Rocket.Chat (`registerGuest`), so Rocket.Chat's own (untranslated) form is
skipped. The browser keeps a random visitor id (`chat-token` in local storage) so that a returning visitor stays
the same contact in Rocket.Chat.

The chat window's title bar shows the company name of the page language (`company` in `site/hugo.toml`). The
Rocket.Chat widget itself has no Greek translation, so on Greek pages the few remaining widget texts (message
field placeholder, "Options", system lines) are in English; the field `language` in the chat still says `el`, so the
agent knows to answer in Greek.

Rocket.Chat's own greeting messages (*Omnichannel → Livechat Triggers*, "AVATAX greeting DE/EL/EN") are
**disabled**: they added an extra card and two more clicks, and Rocket.Chat does not keep trigger messages in the
chat after the visitor has filled in the form. If triggers are ever used again, give each one a *Visitor page URL*
condition for AVATAX pages only, e.g. `^https?://((www\.|admin\.|new\.)?avatax\.eu|localhost:8090)/de/` (the server is
shared with the other website). The CSS rule for `.rocketchat-widget[data-state="triggered"]` in
`site/assets/css/main.css` stays as a safeguard for that case.

#### Chat button inside a page (e.g. on a service page)

Besides the corner button, a page can contain its own chat button, for example below the service description.
Write this on its own line in the page text (`site/content/…/name.xx.md`), with an empty line before and after:

```markdown
{{< chat >}}
```

This shows the standard label ("Questions? Chat with us" / "Fragen? Chatten Sie mit uns" / "Ερωτήσεις;
Συνομιλήστε μαζί μας", in `site/i18n/*.toml` as `chatAsk`). For a label of your own, put it in quotes:

```markdown
{{< chat "Fragen zum Zuzug? Chatten Sie mit uns" >}}
```

Example: `site/content/services/relocation.de.md`, below the list "So unterstützen wir Sie".

- The button behaves like the corner button: it appears only while an agent is *Available*, and a click opens the
  short privacy notice with "Start chat" (or the running chat). Nothing is loaded from the chat server before that.
- **Topic:** the chat records which page the visitor started it from in the field `topic` (Room Information, ⓘ).
  By default this is the page's `linkTitle` (e.g. "Steuervorteile beim Zuzug"). To use another text, add
  `chatTopic: "…"` to the page's front matter; it then also applies to the corner button on that page.
- Each language is a separate file: to have the button on the English and Greek page as well, add the line there too
  (with a label in that language).
- When the live chat is switched off (`rocketchatURL` empty), the line produces nothing.

---

## 9. Check and publish

```bash
make preview        # build the final website locally (http://localhost:8090)
make test           # automatic checks: broken links, old addresses, forms
git add -A
git commit -m "Short description of the change"
git push            # copy on GitHub (backup)
make deploy         # publish on the server
```

`make preview` stops the live-reload server and shows the finished website as it will appear online.
To continue editing afterwards, run `make dev` again.

If something goes wrong after publishing:
```bash
ssh sysop@sigalas.eu 'cd /home/sysop/docker_data/go_avatax && make rollback'
```
This restores the previous version immediately.

---

## 10. Quick help

| Problem | Solution |
|---|---|
| Page shows an error after saving | Usually a missing `"` or wrong indentation in the settings block. Run `docker compose logs hugo` to see the file and line |
| Change not visible | Wait 2 seconds and reload the browser (Ctrl+Shift+R) |
| `make dev` doesn't start | Run `make down`, then `make dev` again. Is Docker running? (`systemctl status docker`) |
| "permission denied" for `public` or `logs` | `sudo chown -R "$(id -u):$(id -g)" public* logs`, then `make dev` again |
| Which containers are running? | `docker compose --profile dev ps` |
| Form messages don't arrive (server) | `ssh sysop@sigalas.eu 'cd /home/sysop/docker_data/go_avatax && docker compose logs formsvc'` |
