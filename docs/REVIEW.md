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

## Pages to proofread

- `careers/apply.de.md` — "neue deutsche Seite, aus dem Griechischen übersetzt"
- `careers/apply.en.md` — "new English page, translated from EL"
- `careers/_index.de.md` — "neue deutsche Seite"
- `careers/_index.el.md` — "νέο εισαγωγικό κείμενο"
- `careers/_index.en.md` — "new English page"
- `careers/logistis-a-taxeos.el.md` — "ad from 2020 – still open?"
- `careers/voithos-logisti.el.md` — "ad from 2020 – still open? Added «(για τους άνδρες υποψηφίους)» to the military-service requirement"
- `_index.de.md` — "neue Texte der Startseite"
- `_index.el.md` — "νέα κείμενα αρχικής σελίδας"
- `_index.en.md` — "new home page texts"
- `privacy.de.md` — "ENTWURF – vor Livegang von der Gesellschaft / einem Rechtsberater prüfen lassen (Datenschutz-E-Mail-Adresse ergänzen)"
- `privacy.el.md` — "ΠΡΟΣΧΕΔΙΟ – πρέπει να ελεγχθεί από την εταιρεία / νομικό σύμβουλο πριν από τη δημοσίευση (προσθήκη e-mail για θέματα προσωπικών δεδομένων)"
- `privacy.en.md` — "DRAFT – must be checked by the company / a legal advisor before going live (add a data-protection e-mail address)"
- `services/payroll.de.md` — "neue Leistungsseite – abgeleitet aus den Aufgaben in den Stellenanzeigen; bitte Umfang bestätigen"
- `services/payroll.el.md` — "νέα σελίδα υπηρεσίας – βασισμένη στις αρμοδιότητες των αγγελιών· επιβεβαιώστε το εύρος"
- `services/payroll.en.md` — "new service page – based on the tasks in the job ads; please confirm the scope"
- `services/start-in-greece.el.md` — "νέα ελληνική σελίδα, μετάφραση από DE/EN"
