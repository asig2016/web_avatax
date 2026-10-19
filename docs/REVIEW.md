# Content review before going live

Pages with a `review:` field in their front matter contain **new or translated text** that someone from the
firm has to read and approve. After approving a page, delete its `review:` line.
List the remaining pages with: `grep -rn '^review:' site/content`

## Decisions and open points – please confirm

1. **No link to other companies:** the site deliberately mentions no partner or sister company.
2. **Payroll page (new):** "Payroll and HR administration" was derived from the tasks in the job ads
   (hirings, departures, payroll, filings). The old site did not list payroll as a separate service. Please confirm.
3. **Accounting page:** mentions bookkeeping "including the electronic books (myDATA)" and financial statements
   "prepared and signed by our licensed accountants (class A)", as on the old site. Please confirm both.
4. **Correction:** the old German text said the accountants help "bei den Wirtschaftsprüfungen" (statutory audits).
   The Greek original means **tax audits**, so it now says "steuerliche Betriebsprüfungen".
5. **Job ads** (Senior Accountant, Assistant Accountant) are from 2020. Are they still open? (`active: true/false`)
   In the assistant ad, "(Γ' κατηγορίας)" was removed, because book categories no longer exist, and
   "(για τους άνδρες υποψηφίους)" was added to the military-service requirement. The old CV form also listed
   "Στέλεχος Γραμματείας", which had no ad, so it is not included.
6. **Privacy policy** (3 languages) is a draft. It must be checked legally, and a data-protection e-mail address
   must be added.
7. **E-mail address:** the site shows only the phone number (+30 211 1036100). Which address receives the forms?
   (`MAIL_TO_CONTACT` / `MAIL_TO_CV` in `.env` on the server)
8. **Member logo:** EEA – Επαγγελματικό Επιμελητήριο Αθηνών (current logo from eea.gr). Is the membership still current?
9. **No industries page** (owner decision). The old list was identical to that of another firm. The old URLs
   redirect to About us.
10. **No team page** for now. It can be added later in the same way as in the sister project (data file + photos).
11. **Certificate:** on 2026-10-01, https://avatax.eu delivered a certificate for **avatax.gr**. Check this at deployment.
12. **Live chat (Rocket.Chat, 2026-10-04):** chat button on all pages, shown only while an agent is available, server
    `https://egroupware.sigalas.eu`, department `AVATAX`. On the server add `CHAT_URL` and `CHAT_DEPARTMENT` to `.env`
    (see `.env.example`). Before going live: allow framing of `/livechat` on that
    server, set up the department and custom fields (`docs/MANUAL.md` § 8), approve the new chat section of the privacy
    policy and the chat texts in `site/i18n/*.toml`. Note: the chat server's host name is visible in the page source
    (CSP); check that this is acceptable under the "no other company" rule.
13. **Legal notice (new, 2026-10-04):** `legal.{en,de,el}.md` (`/legal-notice/`, `/de/impressum/`,
    `/el/nomika-stoixeia/`), linked in the footer and in the chat notice. Greek law (π.δ. 131/2003 Art. 4) and EU
    company law require these details on the website; please fill in the data in [brackets]: e-mail address,
    Γ.Ε.ΜΗ. number, VAT number, Δ.Ο.Υ., legal representative and role, registration number of the firm with the ΟΕΕ.
    The liability note deliberately does **not** exclude liability: under Art. 332 AK (licensed profession) and
    § 309 No. 7 BGB such an exclusion would be void. It only states that the information is general, that answers
    before an engagement (website, chat, e-mail, phone) are non-binding, and that binding advice requires a written
    engagement. Also confirm that AVATAX holds the rights to the photos (copyright section). The texts should be
    checked by a lawyer.
14. **Relocation page split (2026-10-05):** "Tax incentives for moving to Greece" is now an overview
    (`services/relocation/_index.*.md`, same URLs as before) with two sub pages: 7% flat tax for pensioners
    (Art. 5B, `pensioners.*.md`) and 50% exemption for employees and freelancers (Art. 5C, `employees.*.md`). The
    text was moved, not rewritten; new are the overview intro, the page intros and SEO fields. "Formalities" and
    "How we support you" appear on all sub pages. Please check the new pages (`review:` lines).
    New third sub page **€100,000 flat tax for investors (Art. 5A, `investors.*.md`)**, researched from Art. 5A as
    amended by Law 5313/2026 and AADE decision A.1147/2026. The same law moved the 5B payment date from July to
    **December** and abolished the 31 March application deadline; the pensioners page was corrected accordingly.
15. **"Your start in Greece" split (2026-10-05):** overview (`services/start-in-greece/_index.*.md`, same URLs) with
    four sub pages: **without an establishment** (new, researched: VAT registration/OSS/reverse charge, permanent
    establishment risk, posting notification and A1, employing staff without a company, commercial agents),
    representative office, branch, subsidiary (existing text plus typical tasks, accounting, I.K.E./A.E.
    overview with A.E. minimum capital €25,000). Please check especially the new page and the DE comparisons
    (§ 12 AO, OSS via BZSt, agent's compensation like in German law) and the EL references (π.δ. 219/1991,
    ν. 4072/2012, ν. 4548/2018).
    Then split further into **companies** (`companies/`, five sub pages) and **private individuals**
    (`private-individuals/`). New, researched: **foreign company holding property** (special real estate tax ΕΦΑ
    15% of objective value per year, exemptions on disclosure of shareholders down to natural persons, annual
    declaration and exemption application, A.1014/2026) and **buying property as a private individual** (AFM, bank
    account, justification of funds, notary/lawyer/engineer, cadastre, transfer tax 3.09%, VAT on new buildings,
    ENFIA, private use as a holiday home, rental taxation from 15%, short-term rental register ΑΜΑ, Golden Visa
    without amounts). Please check both pages.
    Added 2026-10-06 for private individuals: **staying longer in Greece on a rental** (registration certificate,
    lease declaration and tenant's tax number, tax residence after 183 days with the exception for tourist and
    similar stays up to 365 days under Art. 4 Income Tax Code, EHIC/S1) and **self-employed in Greece** (start of
    activity, e-EFKA, myDATA, VAT small-business scheme without amounts, minimum presumed income with exceptions,
    link to the 50% exemption). Please check.

16. **Renting out property (new page, 2026-10-10):** `services/start-in-greece/private-individuals/renting-out.*.md`,
    written from research, not from the law text itself. Please check: the rental income scale for income from 2026
    (15/25/35/45%, Art. 40(4) KFE as amended by Art. 8 Law 5246/2025); the 5% flat deduction and its link to bank
    payment (Art. 210 Law 5222/2025); short-term rental (under 60 days, up to two properties = property income,
    three or more = business activity with VAT, levy per night); the company section (22% corporate tax, 5% on
    dividends, transfer tax on contribution of a property); DE page: treaty exemption in Germany and
    § 32b(1) sentence 2 no. 3 EStG. The EL page addresses Greeks living abroad. Photos of a client's villa (`start-home.jpg`, `start-stay.jpg`) are used on the owner's
    statement that the client agreed; written consent should be on file.

17. **ETL GLOBAL membership (2026-10-10):** stated in the footer ("Member of", with the network's red logo taken from etl-global.com) and in a new
    section on the About page (en/de/el). Exception to hard rule 1, recorded in `AGENTS.md`. Please check: the exact
    name and link the network wants members to use, whether the logo file and its use match the network's rules for members, and the description "network of independent advisory firms".

18. **Λογιστικός Σύλλογος Αθηνών (2026-10-10):** added to the footer under "Member of" on the owner's instruction.
    The seal was taken from lsa.gr, where only a white and a pale grey version are published; the footer uses the
    grey one recoloured dark for the white badge. Please confirm the membership wording and replace the file with
    an official dark version if the association provides one.

## SEO extension (2026-10-01) – please check the facts

The service pages were extended with "who it is for", "how we work" and FAQ sections. Please confirm especially:
- **Tax consultancy:** the section on the **7% flat tax for foreign pensioners (Art. 5B Income Tax Code)**. Do you
  offer this (eligibility check, application, annual returns)? Also "E1 / E9 returns for private persons".
- **Fiscal representation:** mandatory for non-EU companies, joint liability "as a rule", typical documents
  (notarised power of attorney with apostille, Greek translation).
- **Payroll:** ERGANI II and the digital work card, annual payroll certificates.
- **Accounting:** the four-step process, and taking over books from a previous accountant during the year.

## Payroll page and new relocation page (2026-10-01)

- **Payroll** was rewritten into "Payroll, labour law and social security": advice on social security, permissible
  contract arrangements and working-time arrangement (διευθέτηση χρόνου εργασίας); **digital work card checks
  "under a separate agreement"**; employing staff in Greece without a Greek company (from the German info/offer
  document; fees and names were deliberately **not** published). Please confirm the scope, especially the
  "client account" (Treuhandkonto) for payments.
- **New page "Moving to Greece (Art. 5B / 5C)"** based on the two German information sheets. Corrections from
  current research: the 5B application deadline in the sheet (31 March) is outdated; for 2026 it was extended to
  **31 October** (documents by 30 November), so the site does not state fixed deadlines. The 5C sheet (05.2021)
  contained one-off 2021 deadlines that were left out. The solidarity contribution is not mentioned (largely abolished).
  The sheets are signed with the name of a person from another firm; no names were taken over.

## Audience-specific rework of accounting, tax, fiscal representation (2026-10-01)

Greek pages address Greek clients; German/English pages address foreign companies and individuals. Facts were
checked against sources. Please verify especially:
- **Fiscal representation:** the two roles are now distinguished. The **tax representative** (Art. 8 Law 5104/2024)
  is *not* liable and can be an individual or a legal entity. The **VAT fiscal representative** (VAT Code, Law
  2859/2000) is mandatory for non-EU companies and *jointly liable*. Per Art. 8 Law 5104/2024 and AADE decision A.1069/2024 (amended by A.1125/2024), the tax
  representative is **optional** (no need if AADE notifications are accepted at the declared contact details), can be a
  **natural or legal person**, and a foreign company that appoints none must register its legal representative in the
  tax register. The German employment info/offer ("must be a natural person") is outdated on this point.
- **Accounting:** e-invoicing mandatory since 2 Feb 2026 for revenue > EUR 1m; the date for all other businesses
  is reported differently (1 Oct vs 2 Nov 2026), so no fixed date is given. IFRS thresholds (5%) per the IFRS
  jurisdiction profile.
- **Tax (DE/EN):** treaty Germany–Greece (1966): statutory pensions generally taxed in Greece, civil-service pensions in
  Germany. Property transfer tax "usually 3% plus municipal surcharge". No rental or inheritance tax rates are given,
  because they changed recently.
- **Tax (EL):** 30% electronic expenses rule with 22% tax on the shortfall; minimum imputed income for self-employed.

## Home page text rework (2026-10-02)

- The intro now says what the firm does instead of general statements ("built on trust", "clients are partners").
- The facts strip ("4 eyes", "Class A", languages) was removed because it repeated the pillars. "Class A" is now a
  pillar ("Licensed accountants"). Please confirm the claim, see point 3.
- "What sets us apart" became "How we work", and "Who we work for" became "Our clients".
- Tax representative (Art. 8 Law 5104/2024) and VAT fiscal representative (Law 2859/2000) are now named separately.
- Link labels on the home page are now the same as the menu labels (`linkTitle`).
- DE: "Stammhaus" in place of "Zentrale". Owner decision 2026-10-02: the firm is described as **"Buchhaltungs- und
  Steuerberatungsgesellschaft griechischen Rechts"** (home intro and footer `legalForm`). "griechischen Rechts"
  makes clear that it is not a German Steuerberatungsgesellschaft under § 53 StBerG.
- EL: more technical terms (ΕΛΠ/ν. 4308/2014, myDATA, ΑΠΔ, ΕΡΓΑΝΗ ΙΙ, Ε1/Ε2/Ε9, άρθρα 5Β/5Γ ΚΦΕ).

## Review and conversion rework (2026-10-16) – please check

Independent review of the German site (conversion, positioning, search); batches 1 + 2 implemented.

- **„Steuerberater":** not used as our title any more (tax page seoTitle now „Steuerberatung in Griechenland");
  the tax page explains descriptively that the Greek profession is Λογιστής-Φοροτεχνικός – **legal check of this
  sentence** recommended.
- **Contact flow:** „Nächste Schritte" on every service page and on About; contact page with the full version.
  Redesigned 2026-10-17 after the owner's mock-up; no price in this block or on the contact page any more.
  Please confirm that the phone hours „Mo–Fr 9–17 Uhr … auf Deutsch" are always covered.
- **„Über uns" split into three pages (2026-10-17):** „Über AVATAX" (rewritten: who we are, philosophy, team, clients),
  new page **„Unsere Arbeitsweise"** (`how-we-work.*.md`, /de/arbeitsweise/) and the fees page (with „Beginn des
  Mandats" and the note on additional costs). Please proofread; EN/EL About pages restructured the same way on 2026-10-18.
- **New page „Honorare & Zusammenarbeit"** (`fees.*.md`, /de/honorare/, in the main menu, 2026-10-17): fee models,
  free first meeting for companies, first consultation for private individuals 200 € zzgl. 24 % USt. = 248 €
  (not credited). **Please check** which fee model is named for which service (Buchhaltung/Lohn monthly flat fee,
  Steuerberatung usually time-based) and what the consultation covers („Enthalten" / „Nicht enthalten"). Note: the disabled online booking still labels
  appointments „kostenfrei" (`bookFree`) – adjust before switching it on again.
- **Contact form:** phone optional, new field „Unternehmen", subject replaced by the choice „Ihr Anliegen".
- **FAQs** only on the three relocation pages; useful answers of the others moved into the text.
- **Home, About, Ihr Start, Selbständig, Länger in Griechenland:** rewritten (German angle first, no self-praise,
  business-first openings). Please proofread.
- **All languages active again (2026-10-17):** English at `/`, German at `/de/`, Greek at `/el/`.
  Temporary redirects to German have been removed; language switches and localized 404 pages are active.
- **EN/EL parity (2026-10-18):** the German review changes were carried over to the English and Greek pages,
  adapted to their audiences (rule 4): FAQs only on the three relocation sub pages (answers moved into the text),
  `ctaTopic` on every service page, About in the new structure with „Nächste Schritte", home pillars and client
  cards as in DE (four-eyes review by *a second professional*; financial statements **and** tax returns signed by
  licensed accountants), order 5A/5B/5C, „Your start in Greece" first in the services menu, linked company list,
  factual long-stay and self-employed openings, new titles/seoTitles for search (e.g. „Greek Non-Dom Regime").
  Every changed page has a `review:` note dated 2026-10-18. **Please check in particular:**
  - **EN/EL VAT fiscal representative:** both pages still described the old rule (Law 2859/2000: every non-EU
    company). They now follow the German page: Art. 41(3)(d) Law 5144/2024, only for companies established
    neither in Greece nor in another EU state that owe the VAT themselves; not needed under reverse charge.
  - **EN tax page:** new paragraph for searchers („Looking for a tax adviser in Greece? …") like the German one –
    legal check of the sentence on the profession.
  - **„Accounting department support" (EN/EL):** still active in EN and EL (German page deactivated); FAQ merged
    into the text. The EN home card no longer links to it (as in DE), the EL home card still does (Greek companies).
    Decide whether the EN page should be deactivated like the German one.
  - **EL legal details taken over from the old FAQs:** single-entry threshold €1.5m (accounting), working-time
    and probation rules (payroll), e-payment rule 30 %/22 % (tax); the article number of the VAT small-business
    scheme was removed from the self-employed page because Law 5144/2024 renumbered the VAT Code.
  - **EL wording:** home pillar „Σταθερός συνεργάτης" (was „υπεύθυνος επικοινωνίας"), subsidiary title
    „Σύσταση θυγατρικής", „θεώρηση γνησίου υπογραφής" for the power of attorney.
- **Server (owner):** https redirects of avatax.gr / avatax.net / www.avatax.eu to avatax.eu, `client_max_body_size
  6m` and resolver in the reverse proxy; logrotate for `logs/nginx` (privacy policy: 30 days).
- **Later (batch 3):** deeper B2B pages (subsidiary, branch, representative office, without establishment),
  sections on reporting to the parent company, tax audits and transfer pricing, verified Greece/Germany comparison
  tables, „Stand" dates, new pages (VAT registration, posting staff, German pension, selling property,
  inheritance, Golden Visa taxes), Google Business Profile.

## Pages to proofread

- `careers/apply.de.md` — "neue deutsche Seite, aus dem Griechischen übersetzt"
- `careers/apply.en.md` — "new English page, translated from EL"
- `careers/_index.de.md` — "neue deutsche Seite"
- `careers/_index.el.md` — "νέο εισαγωγικό κείμενο"
- `careers/_index.en.md` — "new English page"
- `careers/logistis-a-taxeos.el.md` — "ad from 2020 – still open?"
- `careers/voithos-logisti.el.md` — "ad from 2020 – still open? Added «(για τους άνδρες υποψηφίους)» to the military-service requirement"
- `_index.de.md` — "Startseite am 2026-10-02 sachlicher formuliert (ohne Kennzahlen-Leiste)"
- `_index.el.md` — "αναδιατύπωση αρχικής σελίδας 2026-10-02 (πιο τεκμηριωμένο ύφος, χωρίς λωρίδα στοιχείων)"
- `_index.en.md` — "home page texts rewritten 2026-10-02 (more factual register, no facts strip)"
- `privacy.de.md` — "ENTWURF – vor Livegang von der Gesellschaft / einem Rechtsberater prüfen lassen (Datenschutz-E-Mail-Adresse ergänzen)"
- `privacy.el.md` — "ΠΡΟΣΧΕΔΙΟ – πρέπει να ελεγχθεί από την εταιρεία / νομικό σύμβουλο πριν από τη δημοσίευση (προσθήκη e-mail για θέματα προσωπικών δεδομένων)"
- `privacy.en.md` — "DRAFT – must be checked by the company / a legal advisor before going live (add a data-protection e-mail address)"
- `services/accounting.de.md` — "Text für deutschsprachige Unternehmen auf Basis von Quellen (Ges. 4308/2014, AADE, IFRS-Profil Griechenland) – bitte prüfen"
- `services/accounting-department-support.de.md` — "SEO-Erweiterung (neue Abschnitte und FAQ) – bitte prüfen"
- `services/accounting-department-support.el.md` — "επέκταση SEO (νέες ενότητες και FAQ) – ελέγξτε"
- `services/accounting-department-support.en.md` — "SEO extension (new sections and FAQ) – please check"
- `services/accounting.el.md` — "κείμενο για Έλληνες πελάτες βάσει πηγών (ν. 4308/2014, ΑΑΔΕ) – ελέγξτε"
- `services/accounting.en.md` — "text for foreign companies based on sources (Law 4308/2014, AADE, IFRS profile Greece) – please check"
- `services/fiscal-representation.de.md` — "Text für ausländische Mandanten auf Basis von Quellen (Art. 8 Ges. 5104/2024, griechisches UStG Ges. 2859/2000) – bitte prüfen"
- `services/fiscal-representation.el.md` — "κείμενο για Έλληνες πελάτες βάσει πηγών (άρθρο 8 ΚΦΔ, ΚΦΠΑ) – ελέγξτε"
- `services/fiscal-representation.en.md` — "text for foreign clients based on sources (Art. 8 Law 5104/2024, Greek VAT Code Law 2859/2000) – please check"
- `services/payroll.de.md` — "Text für deutschsprachige Unternehmen, die erstmals in Griechenland Mitarbeiter beschäftigen – bitte Umfang und Formulierung prüfen"
- `services/payroll.el.md` — "κείμενο για Έλληνες πελάτες (ορολογία, νομοθεσία) – ελέγξτε εύρος υπηρεσιών και διατυπώσεις"
- `services/payroll.en.md` — "text for foreign companies employing staff in Greece for the first time – please check scope and wording"
- `services/relocation.de.md` — "neue Seite auf Basis der Informationsblätter zu Art. 5B/5C (Stand 01.2024 bzw. 05.2021) und aktueller Recherche – bitte fachlich prüfen"
- `services/relocation.el.md` — "νέα σελίδα βάσει των ενημερωτικών σημειωμάτων για τα άρθρα 5Β/5Γ και επίκαιρης έρευνας – ελέγξτε"
- `services/relocation.en.md` — "new page based on the information sheets on Art. 5B/5C (status 01.2024 / 05.2021) and current research – please check"
- `services/start-in-greece.de.md` — "SEO-Erweiterung (neue Abschnitte und FAQ) – bitte prüfen"
- `services/start-in-greece.el.md` — "νέα ελληνική σελίδα, μετάφραση από DE/EN"
- `services/start-in-greece.en.md` — "SEO extension (new sections and FAQ) – please check"
- `services/tax.de.md` — "Text für deutschsprachige Mandanten auf Basis von Quellen (DBA 1966, griechisches Steuerrecht) – bitte prüfen"
- `services/tax.el.md` — "κείμενο για Έλληνες πελάτες βάσει πηγών – ελέγξτε"
- `services/tax.en.md` — "text for foreign clients based on sources – please check"

## Contact wording review — 2026-10-17

Human review pending for the updated DE/EN/EL contact steps, phone-hour labels, message guidance and
confirmations (`site/data/next_steps.toml`, `site/i18n/*.toml`), plus the fiscal-representation page's
closing headline, introduction and button labels. Changed content pages have `review:` markers.
Scope and fees are agreed before paid work; contact areas keep their fees links and make no response-time promise.

Owner clarification 2026-10-17: a written offer and the client’s written acceptance precede every
appointment, including first meetings. Contact steps and fees pages reflect this in DE/EN/EL;
the free first meeting for companies and the stated private consultation fee remain unchanged.

2026-10-17: fee links now appear on the terms within the contact steps throughout the site
(DE/EN/EL); separate fee boxes removed. The fees pages omit the link to themselves.

2026-10-17: contact-entry buttons now consistently use “Unverbindlich anfragen” (EN/EL adapted),
including home-page buttons, sidebars and closing blocks. Long page-specific labels removed;
existing topic preselection retained. Form submit, navigation and chat labels keep their respective actions.
Human review pending for the new shared DE/EN/EL label.
