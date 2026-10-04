# Invisible proof-of-work captcha for the contact and CV forms

## Status / decision

**Deferred** (owner decision 2026-10-04). Nothing is implemented. Wait until the live site shows real spam,
then decide using the formsvc logs:

```bash
docker compose logs formsvc | grep -c "honeypot hit"     # bots that fill in the hidden field
docker compose logs formsvc | grep -c "token rejected"   # posts without a valid token or sent too fast
docker compose logs formsvc | grep "mail sent"           # what actually got through
```

- **Only a few junk mails get through:** no change needed.
- **Recurring spam mails are getting through:** first add only the single-use tokens (step 3 below, about 30 lines of Go).
  Add proof-of-work (PoW) only if that is not enough.
- **Spam comes from many IP addresses with real browser behaviour:** a captcha barely helps. Use the mailbox's spam
  filter instead.

## Context

The forms (`/api/contact`, `/api/cv`) are already protected by:

- a honeypot field (`website`)
- a signed time token from `/api/token` (HMAC, at least 3 s fill time, at most 180 min old)
- a per-IP rate limit (5 per 10 min)
- JS being required: without JS, the token is empty, so the post is rejected and `<noscript>` shows the phone number

AGENTS.md rule 12 forbids reCAPTCHA and other third-party requests. If more protection is needed, the chosen
option is an **invisible, self-hosted PoW captcha** in the style of ALTCHA. It uses Go stdlib and vanilla JS only.
There are no cookies, no third party and no visible UI. Each token can be used only once.

## Why it helps

- Today, a bot can call `GET /api/token`, wait 3 s and post, with no JavaScript and no browser. Each post costs it
  almost nothing.
- PoW makes every post cost a fraction of a CPU-second, and the token cannot be reused. Mass spam becomes
  expensive. A real visitor doesn't notice, because the puzzle runs while they type.
- Single-use tokens close a replay hole: today one token can be posted many times within 3 h, across many IPs.

## Trade-offs

- It does not stop a determined attacker running a real browser, or human spammers. The honeypot, rate limit and
  mailbox filter still handle those.
- It costs the visitor some CPU and battery (see below).
- It adds about 150 lines of code (Go + JS) to maintain.
- The used-token list is lost on a restart. That is harmless because tokens expire anyway. It only works with a
  single formsvc instance, which is the current setup.

## Old phones

- **Page loads are not affected.** The puzzle starts only when someone clicks or types in a form.
- **It runs while the visitor types**, which takes at least 20–60 s. Estimated solve time: well under 0.1 s on a recent
  phone, about 0.5–1 s on an old cheap Android. It is almost always finished before "Send".
- **The phone stays responsive.** The work is split into small pieces, so typing and scrolling are not blocked.
- **Bad luck makes a solve slower.** The number of attempts varies randomly, so on about 1 in 100 attempts it takes
  about 4–5 times longer. On a very old phone that is roughly 3–5 s, only noticed as a longer "Sending…".
- **`POW_BITS` sets the difficulty.** One step lower is 2 times easier, two steps lower 4 times easier.
- These figures are estimates. Measure with Chrome DevTools CPU throttling before choosing the default.

## Design

The PoW is folded into the existing token, so the form and the flow barely change.

1. **Token** (`formsvc/main.go`, `newToken`/`sign`/`checkToken`): the format changes from `ts.sig` to `ts.salt.sig`.
   - `salt` = 8 random bytes in hex. `sig` = HMAC(`ts.salt`).
   - `/api/token` returns `{"token": "...", "bits": N}`.
2. **Puzzle**: find a `nonce` (decimal) such that `SHA-256(token + ":" + nonce)` has at least N leading zero bits.
   - New config `POW_BITS` (env, default 16, which is about 65k hashes on average).
   - The server needs a single hash to verify.
3. **Server check** (`checkToken` gets the nonce from the form field `pow`):
   - signature check, then the min/max age checks as today, then the leading-zero-bits check
   - **single use**: keep a `map[string]time.Time` of accepted token signatures under a mutex. A replay is rejected
     with the existing `spam` code. Entries older than `MaxTokenAge` are pruned during the call, using the same
     "every 1000 calls" pattern as `rateLimiter.allow`.
   - The token is marked as used only after all validation passes, just before sending, so a user who fixes a field
     error can resubmit. To do this, `common()` checks the token, and a new `s.consume(tok)` call before `s.send`
     marks it as used.
4. **Client** (`site/assets/js/main.js`):
   - Add a compact synchronous SHA-256 implementation (~40 lines, no dependency). `crypto.subtle` is too slow per call
     for tens of thousands of hashes.
   - `fetchToken` stores the token and bits. A `solve(form)` function then starts on the first `focusin`/`input` in
     the form, not on page load. It runs in chunks of about 5k hashes with `setTimeout(0)` between them and writes
     `form.elements.pow.value`.
   - On submit: `await` the pending solve promise. This also covers a token re-fetch after a `spam` error.
5. **Form partial** (`site/layouts/_partials/form-common.html`): add `<input type="hidden" name="pow" value="">`
   next to `token`.
6. **CSP / privacy**: no change needed (same origin, no worker, no cookies).

## Files

- `formsvc/main.go`: config `PowBits`, token format, PoW verification, used-token set, `consume` before `send` in
  both handlers
- `formsvc/main_test.go`:
  - `tokenAt` also returns a solved nonce; tests use `PowBits: 8` for speed
  - new cases: missing nonce, wrong nonce, replayed token, resubmit after a validation error is still accepted
- `site/assets/js/main.js`: SHA-256 and solve logic, wired into the existing `fetchToken`/submit flow
- `site/layouts/_partials/form-common.html`: hidden `pow` field
- `.env.example`, `docs/DOCKER.md` (env table): document `POW_BITS`

## Verification

1. `make test`: go vet and test, including the new PoW and replay tests.
2. `make dev`, then submit the contact and CV forms at http://localhost:8090:
   - the mail arrives in mailpit at :8026
   - the DevTools network tab shows the `pow` field, and nothing is loaded from third parties
3. Replay: run `curl -X POST` with a captured token+pow twice. The second call returns `400 {"error":"spam"}`.
4. Without `pow`, or with a wrong nonce, the response is `spam`.
5. Measure the solve time with CPU throttling 4–6× and adjust the default `POW_BITS` if it is clearly above about 1 s.
