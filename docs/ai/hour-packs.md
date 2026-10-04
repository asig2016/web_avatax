# Prepaid hour packs (online purchase)

## Status / decision

**Plan only** (owner request 2026-10-04). Nothing is implemented. Before starting, the owner decides the points
under "Owner decisions". The payment provider is still open; the design below works with **Viva Wallet Smart
Checkout** or **Stripe Checkout** (one adapter each, only the chosen one is built).

## Context

Visitors (mainly DE/EN audience: foreign companies and individuals starting in Greece) should be able to buy a
pack of advisory hours online, e.g. 5 h / 10 h / 20 h, and pay by card. AVATAX then issues the invoice (myDATA)
and books the hours against the client in its time tracking.

Constraints from AGENTS.md:

- Rule 12: no third-party requests by default. The payment provider is contacted **only after the visitor
  clicks "Proceed to payment"** (redirect to the hosted checkout, no provider script on our pages).
- formsvc is Go stdlib only and keeps **no state on disk**. Order data travels in the provider's order metadata.
- Content bind-mounted, three languages, `review:` lines, SEO fields, no fixed dates that change.
- The site never sees card data (hosted checkout, no PCI scope beyond SAQ A).

## Flow

```
pack page ──click──▶ order form (our site) ──POST /api/order──▶ formsvc
                                                    │ validates, price from hourpacks.json,
                                                    │ creates checkout session at provider (server-to-server)
         ◀──────────── JSON {url} ──────────────────┘
browser ──location.assign(url)──▶ provider hosted checkout ──paid──▶ /thanks-order/ on our site
provider ──webhook POST /api/payment-webhook──▶ formsvc: verify signature / re-fetch transaction
                                                    └─▶ mail to office (order + invoicing data)
                                                    └─▶ confirmation mail to client
```

- The browser uses `fetch` + `location.assign`, not a plain form redirect: CSP `form-action 'self'` would block a
  POST that redirects to the provider. The CSP itself needs **no** change (no provider script, frame or connect).
- Only the webhook counts as proof of payment. The return page only says "payment received, confirmation follows
  by e-mail" and is `noindex`.
- Without JS the order form shows the phone number and the contact page (same pattern as the existing forms).

## Single source for packs and prices

`site/data/hourpacks.json` (JSON, because formsvc must parse it with stdlib):

```json
{ "currency": "EUR", "vatRate": 24, "validityMonths": 12,
  "packs": [ { "id": "h5",  "hours": 5,  "priceNet": 0 },
             { "id": "h10", "hours": 10, "priceNet": 0 },
             { "id": "h20", "hours": 20, "priceNet": 0 } ] }
```

- Hugo renders the pack cards from it. formsvc reads the same file through a read-only bind mount
  (`./site/data/hourpacks.json:/config/hourpacks.json:ro,z`, env `HOURPACKS_FILE`) and **always takes the price from
  the file**, never from the browser.
- The prices are placeholders until the owner sets them. Shown as gross prices (incl. VAT) for consumers, net + VAT
  line for businesses.

## Implementation steps

1. **Content (EN/DE/EL)**, all with `review:`, `seoTitle`, `description`:
   - `services/hour-packs.{en,de,el}.md`: what a pack covers / does not cover, validity, how hours are recorded
     and reported, link to terms. Service page front matter (`teaser`, `icon`, `menus.main.parent: services`).
     Label leads with the benefit (e.g. "Prepaid advisory hours" / "Beratungsstunden im Voraus" /
     "Προπληρωμένες ώρες συμβουλευτικής").
     EL: short, precise, invoicing terms. DE/EN: explain how it works for clients abroad.
   - `order.{en,de,el}.md` (`layout: order`): the order form.
   - `thanks-order.{en,de,el}.md` and reuse `form-error`.
   - `terms.{en,de,el}.md`: terms for hour packs (validity, refund of unused hours, withdrawal right, which services
     count). Text from the owner / legal advisor; we only provide the structure.
2. **Layouts:** `layouts/order.html` and `_partials/hourpacks.html` (cards from `site.Data.hourpacks`, "Order"
   button → order page with `?pack=h10`). The form follows the existing `form-common.html` pattern
   (honeypot, `/api/token`, `js-form`).
   Fields: pack, name, e-mail, phone, client type (consumer / business), company, country, address,
   ΑΦΜ / VAT ID (required for business; for EU business VIES format check), message, accept terms (required),
   and for consumers the early-start request (see decisions).
3. **formsvc `order.go`:** `POST /api/order`:
   validate like `handleContact` (token, honeypot, rate limit), load pack from `HOURPACKS_FILE`, compute gross,
   create the checkout session through a small interface
   `type checkout interface { Create(order) (redirectURL string, err error); Verify(r *http.Request) (paidOrder, error) }`.
   Order data (pack, invoicing data, language) goes into the provider metadata, so nothing is stored locally.
   Reply `{"ok":true,"url":"…"}`.
4. **Provider adapter (only the chosen one):**
   - **Viva:** OAuth2 client credentials → `POST /checkout/v2/orders` (amount in cents, `customerTrns`,
     `merchantTrns` = our order id, `tags`), redirect to `…/web/checkout?ref=<orderCode>`. Webhook "Transaction
     Payment Created": answer Viva's GET verification key request, then **re-fetch the transaction** via API and
     compare amount and order code before trusting it.
   - **Stripe:** `POST /v1/checkout/sessions` (`mode=payment`, `price_data`, `metadata`, `customer_email`,
     `success_url`, `cancel_url`, `locale`). Webhook `checkout.session.completed`: verify `Stripe-Signature`
     (HMAC-SHA256 with the webhook secret, timestamp tolerance 5 min), check `payment_status=paid` and amount.
   - Secrets only in `.env`: `PAY_PROVIDER`, `PAY_API_KEY` / `PAY_CLIENT_ID` + `PAY_CLIENT_SECRET`,
     `PAY_WEBHOOK_SECRET`, `PAY_MODE=test|live`, `MAIL_TO_ORDERS` (defaults to `MAIL_TO_CONTACT`).
5. **`POST /api/payment-webhook`:** verify → mail to office with all invoicing data and the provider reference
   ("issue invoice / book hours"), confirmation mail to the client in their language (pack, amount, reference,
   "invoice follows"). Idempotent enough for retries: duplicate mails are acceptable (no state), the subject
   carries the provider reference so duplicates are obvious.
6. **nginx:** `/api/payment-webhook` already goes to formsvc through `location /api/`. Exclude it from any
   future captcha / rate limit, limit the body size (64 KB).
7. **Privacy policy (3 languages):** new section "Online payment": provider, data passed on (name, e-mail,
   amount, order data; card data only at the provider), legal basis Art. 6(1)(b) and (c) GDPR (tax retention).
8. **Docs:** `.env.example` (PAY_*), `docs/MANUAL.md` (changing packs and prices in `hourpacks.json`,
   what to do with an order mail), `docs/DEPLOY.md` (webhook URL to register at the provider, switch test → live),
   `docs/REVIEW.md` (terms, prices, legal texts to approve).

## Owner decisions (before implementation)

1. **Provider:** Viva Wallet or Stripe (fees, payout account, contract). Bank transfer only is a different, simpler
   variant: order form → mail with IBAN / IRIS details, no webhook.
2. **Packs:** hours and net prices; validity of unused hours; refund of unused hours.
3. **Scope:** which services count against the hours (e.g. advisory and tax questions, not recurring bookkeeping
   or payroll fees).
4. **Customers:** consumers and/or businesses only. Consumers: 14-day withdrawal right for distance contracts
   (Law 2251/1994, Directive 2011/83/EU); for an earlier start the consumer must expressly request it and
   acknowledge the consequence – wording by the legal advisor. Business-only would avoid this.
5. **VAT:** handling of business clients in other EU countries (reverse charge) and non-EU clients; the order mail
   must carry what the invoice needs.
6. **Invoicing:** invoice issued manually in the existing invoicing software (myDATA) after the order mail, or later
   an automatic link.
7. **Hour tracking:** where the balance lives (e.g. EGroupware project with hour budget / timesheets) and how the
   client sees the remaining hours (monthly report by e-mail is enough at first).

## Verification

- `make test`: Go unit tests for `/api/order` (validation, price from file not from request, metadata) and for the
  webhook (valid / invalid signature, wrong amount, replayed old timestamp) with a fake provider (`httptest`).
- Local: `make dev`, provider **test mode** with test keys in `.env` and the provider's published test cards. Stripe:
  forward webhooks with `stripe listen --forward-to localhost:8090/api/payment-webhook`. Viva: webhooks need a
  public URL → test on staging `new.avatax.eu`.
- Check: no request to the provider before clicking "Proceed to payment" (network tab); no CSP errors; order mail and
  client mail arrive in mailpit in all three languages; cancel on the checkout page returns to the order page.
- Switch to live keys only after the owner approved terms, prices and privacy text.
