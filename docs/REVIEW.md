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
