# Online appointment booking

Status 2026-10-14: built and tested end to end against the dev5 EGroupware, **not deployed, nothing
committed**. Payment and invoicing (phases 3 and 4) are planned, not built.

## What the visitor sees

- `/book/`, `/de/termin/`, `/el/rantevou/` (layout `booking`):
  1. choose a service (topic, duration, price);
  2. choose a person, or "first available" when several offer it;
  3. choose a free time in a week view (Athens time);
  4. enter name, e-mail, optional phone, company and message, and give consent.
- "Request appointment" holds the slot for 15 minutes and mails a confirm link.
- The confirm page (`/book/confirm/`, `/de/termin/bestaetigen/`, `/el/rantevou/epivevaiosi/`) shows
  the booking. Only the button press confirms it: a mail scanner that opens the link does nothing.
- After confirmation:
  - the event is in the person's EGroupware calendar;
  - the client gets a mail with a cancel link (`/book/cancel/`, `/de/termin/absagen/`, `/el/rantevou/akyrosi/`);
  - depending on the EGroupware setting, the client also gets the calendar's .ics invitation.
- Online cancellation is possible until 24 h before the appointment (EGroupware config). After that
  the page says to call.

## How it works

```
browser ──/api/booking/*──▶ formsvc ──HTTPS, app password──▶ EGroupware /groupdav.php/acbooking/*
          (same origin)     caches catalog 5 min,            app "acbooking": services, persons,
                            slots 30 s; sends the mails       weekly hours, slots, holds, events
```

- **EGroupware is the source of truth.** formsvc stays stateless. The browser never contacts
  EGroupware (rule 12), so the CSP is unchanged.
- **Tokens** (`<id>.<HMAC>`) travel only in the fragment of the mailed links (`#t=…`) and in POST
  bodies, so they never appear in an access log.
- **Spam protection:**
  - the hold uses the same checks as the contact form: token, honeypot, consent, and the form rate limit;
  - EGroupware allows at most 3 open holds per e-mail address;
  - an unconfirmed hold expires.
- **Rate limits:** confirm, cancel and lookup have their own, wider limit (`tokenLimiter`), because
  one visitor makes several of those calls in a row.
- **Free slots** = the person's weekly hours, minus calendar events (not those marked free or
  rejected), minus open holds, with buffers, lead time and horizon applied. Two holds for the same
  slot are serialized by a database lock, and the confirm runs the calendar's own conflict check.

Code:
- `formsvc/booking.go` (+ `booking_test.go`)
- `site/layouts/booking.html`, `booking-token.html`, `_partials/booking-i18n.html`
- `site/assets/js/booking.js`
- `book*.md` content pages and `book*` i18n keys

EGroupware side: app `acbooking` (own repository in the EGroupware tree); see its README.

## Setup (owner)

1. In EGroupware, install `acbooking`.
2. Create the technical user `booking-api`:
   - run rights for *acbooking* and *groupdav*;
   - grant `booking.api` (Admin > Permissions);
   - an app password limited to acbooking + groupdav (Admin > Application passwords);
   - give it an e-mail address of the firm: it is the sender of the calendar invitations.
3. Set up the booking data in EGroupware > Appointment booking:
   - **Services**: title, duration, price, and translations en/de/el;
   - **Bookable persons**: user, public name, services, weekly hours, buffers;
   - **Site configuration**: timezone, hold time, cancel cutoff, prospects addressbook,
     calendar invitation to the client yes/no.
4. Fill in the `.env` block "appointment booking" on the server (see `.env.example`), then run `make update`.
5. Decide whether the booking page goes into the main menu. Right now it is linked from the contact page only.

## Open decisions

- Showing staff names publicly. They come from EGroupware (`public name`), not from this repo.
  Rule 6 is about staff numbers, not names, but names on the web are the owner's choice.
- Which services are offered, and whether any is paid.
- Retention period for booking data (privacy policy section added as draft, `review:` set).
- The calendar invitation mail is in the booked person's language and comes from the technical
  user's address. Turn it off in EGroupware (Site configuration) if the website's own mails are enough.

## Phase 3: payment (planned)

Reuse the design of `hour-packs.md`:
- a hosted checkout behind a provider-neutral `checkout` interface in formsvc;
- the webhook is the only proof of payment.

For a priced service:
1. `hold` returns a checkout URL instead of sending the confirm mail.
2. The booking waits in `pending_payment` (EGroupware already has the status) with a longer expiry.
3. The webhook confirms it via a new `acbooking/holds/{id}/paid`. The paid e-mail doubles as the
   e-mail check.

Refunds on cancellation follow a policy still to be decided.

## Phase 4: acerp invoice (planned)

On payment, acbooking creates an acerp sales document from the service's ERP item code and price.

Open: acerp needs a client and an agreement, so a prospect first has to become an acclients
client. Options: create one per paying prospect, or use one "walk-in" client. To be decided before building it.
