# AGENTS.md – notes for AI agents / developers working on this repo

Website of **AVATAX A.E.** (Accountants & Tax Consultants, Ανώνυμη Εταιρία Λογιστών), Syngrou Ave. 76, Athens.
It replaces the Drupal 7 site at https://avatax.eu. Three languages: **en** (default, at `/`), **de** (`/de/`),
**el** (`/el/`).

Read `README.md` (overview), `docs/DOCKER.md` (containers and commands), `docs/DEPLOY.md` (Hetzner runbook),
`docs/REVIEW.md` (open content points) and `docs/MANUAL.md` (how the owner edits content).

## Hard rules

1. **No mention of or link to any other company** (in particular no sister/partner audit firm). Owner decision
   2026-10-01: the companies stay separate on the web, so that they are not read as an audit "network".
   This includes footers, structured data, texts, images and alt texts.
2. **AVATAX is an accounting and tax firm, not an audit firm.** Bookkeeping, payroll, financial statements and
   tax returns are its core services. Never describe services as statutory audits ("Wirtschaftsprüfung",
   "έλεγχος οικονομικών καταστάσεων"); use "tax audit support" / "Betriebsprüfung" / "φορολογικός έλεγχος".
3. **All three languages.** Every page exists as `name.en.md`, `name.de.md` and `name.el.md`, each with an explicit `url:`
   (Hugo does **not** add the `/de/` or `/el/` prefix to `url`, so write it yourself). New or translated text
   gets a `review: "…"` front-matter line until a human approves it.
4. **Languages have different audiences – no 1:1 translations.** (owner decision 2026-10-01)
   - **EL** pages: Greek clients who know the terminology. Use precise professional terms and legal references
     (ΑΠΔ, ΦΜΥ, ΕΡΓΑΝΗ ΙΙ, ΚΠΚ, ν. 4808/2021 …) and skip basic explanations.
   - **DE/EN** pages: people and companies coming to Greece / doing business in Greece for the first time. Explain
     what is different, start with their questions (e.g. employing staff without a Greek company).
   - **DE** especially: the firm has strong ties to Germany and can compare Greek rules with German ones. Use that.
   - Facts must be consistent across languages; scope and depth may differ.
5. **SEO fields** on every page: `seoTitle` (≤ 60 chars) and `description` (≤ 155 chars). Service pages also get
   `teaser`, `icon` and `menus.main.parent: services`. The H1 comes from `title`, the menu uses `linkTitle`.
6. **No unverifiable claims:** no founding year, no "x years of experience", no staff numbers, no client names.
7. **Old URLs must keep working:** when a URL changes, add a 301 to `nginx/conf.d/redirects.conf` and a line to
   `tests/old-urls.txt`.
8. **Content is bind-mounted, never baked into images.** Project path `/home/asig/docker_data/avatax` on the laptop,
   `/docker_data/go_avatax` on the server (`ssh sysop@sigalas.eu`). Bind mounts use `:z` (Fedora SELinux). Create `public`, `public.next`, `logs/nginx` before the first
   `docker compose up/run`, otherwise Docker creates them as root and Hugo (non-root) cannot write.
9. **Ports:** web `127.0.0.1:8090`, mailpit `:8026`, hugo live reload `:1314`. Another stack of the same kind
   runs on the same laptop and server with 8080/8025/1313. Do not take those ports.
10. **Don't touch the server yourself.** Deployment is done by the owner following `docs/DEPLOY.md`.
11. **Git: never commit and never push on your own.** Only when the owner explicitly says so; commit and push
    are separate permissions. Remote `origin` = `git@github.com:asig2016/web_avatax.git`, branch `master`.
12. **Privacy:** no third-party requests by default (no Google Fonts, no maps embed, no reCAPTCHA). GA4 only
    after consent and only if `ga4ID` is set.

## Writing and content quality rules (owner feedback 2026-10-01)

- **Lists contain one category only.** A list headed "Our services" / "Unsere Leistungen" / "Υπηρεσίες" lists
  services only. Features or benefits (e.g. "online access to your data") go into a sentence below the list,
  not into the list. No separate mini-sections for single service items – merge them into the service list.
- **Consistent list form:** within one list either all items use `**Label:** text` or none; same capitalisation
  and the same grammatical form (all fragments or all sentences).
- **Professional register.** Write like an advisory firm, based on sources – no colloquial headings
  ("Was in Griechenland anders ist" → "Besonderheiten der griechischen Rechnungslegung"), no "so wird schnell klar",
  no boasting ("best possible partner", "considerable sums"), no "we also know…" tone, no exclamation marks.
- **Precise legal terminology.** Don't lump different legal concepts under one loose term in quotation marks.
  Example: Repräsentanz (no permanent establishment while only preparatory/auxiliary), Zweigniederlassung
  (registered in GEMI, permanent establishment, taxable), Tochtergesellschaft (separate legal entity, I.K.E./A.E.).
  "Stammhaus", not „Zentrale". German protected professional titles (e.g. "Steuerberater") are not used as titles.
- **Verify facts against the law text, not against old documents or offers.** Example: Art. 8 Law 5104/2024 +
  AADE A.1069/2024: the tax representative is optional, can be a natural or legal person and is not liable;
  without one, the legal representative of a foreign company registers in the tax register. The VAT fiscal
  representative (Law 2859/2000) is a different role. Never state "must" where the law says "may".
- **No fixed deadlines or dates that change often** (application deadlines, extensions) – say that we check the
  current deadline. Only state dates that are settled in law.
- **Labels lead with the benefit, not the paragraph.** Menu entries, link labels and card titles name what the
  client gets ("Tax incentives for relocating"), not the legal article ("5B/5C"); articles belong in the text.
- **Same term for the same thing** in titles, menu labels, home cards and links within one language.
- **Teasers/descriptions must match the page** after every content change.

## Architecture

The same stack as described in `docs/DOCKER.md`:

- **Hugo 0.154.5** (`hugomods/hugo:exts-non-root-0.154.5`), new template layout (`layouts/home.html`, `page.html`,
  `section.html`, `_partials/`; special pages via `layout:` → `contact`, `careers`, `apply`). Use `.Sites`, not
  `hugo.Sites`.
- **CSS** `site/assets/css/main.css`: brand tokens `--brand` (#6c5020 brown) and `--accent` (#c1b585 gold).
- **formsvc** (Go, stdlib only): `/api/token`, `/api/contact`, `/api/cv`; `SITE_NAME` (in `.env`) appears in mail
  subjects and bodies. The contact handler must accept multipart (FormData) and urlencoded bodies.
- **nginx** container: `default.conf` (CSP is set by Hugo as a meta tag), `redirects.conf` (map;
  `map_hash_bucket_size` stays in default.conf).
- Structured data: `AccountingService` (home, service provider), BreadcrumbList, Service, FAQPage (via `faq:`).

## Commands

```bash
make dev        # laptop: hugo live reload + web + formsvc + mailpit (http://localhost:8090, mails :8026)
make preview    # production build → public/ (keeps public.prev/)
make test       # go vet/test, lychee link check (offline), 301 check of tests/old-urls.txt
make deploy     # rsync to server + `make update` there (owner runs this)
```

## Legal references (checked 2026-10-01)

- Greek Tax Procedure Code = **Law 5104/2024** (replaces Law 4174/2013). Do not cite Law 4174/2013 for current rules.
- Accounting standards: Greek GAAP (Law 4308/2014); electronic books via AADE myDATA.

## Status (2026-10-01)

- Built from the structure of a sister project; content migrated from the old avatax.eu and rewritten in EN/DE/EL.
- Not deployed, nothing committed. Open points: `docs/REVIEW.md`.
