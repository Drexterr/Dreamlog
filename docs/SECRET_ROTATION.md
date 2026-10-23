# Secret rotation runbook

Use this after a suspected leak of Railway (or any host holding production
variables). Treat every secret stored in Railway as exposed. Rotate in this
order; after each step confirm `GET /health` is 200 on the API and the worker
log has no errors, then continue.

**Never paste secret values into chat, tickets or commits.** Set them in the
Railway dashboard (service → Variables) on **both** services (`Dreamlog` = API,
`clever-imagination` = worker) unless a step says API only. Setting a variable
redeploys the service.

Before starting: Railway → Workspace → Audit log (look for variable reads,
unknown members or API tokens), revoke all Railway API tokens, remove unknown
members, and enable 2FA on Railway, GitHub, Supabase, Cloudflare, Upstash,
Google, Anthropic, OpenAI and Azure.

## 1. Database password (`DATABASE_URL`)
1. At your database host (Supabase → Project Settings → Database → Reset
   database password).
2. Rebuild the connection string with the new password and set `DATABASE_URL`
   on both services.
3. Both services reconnect on redeploy. Old sessions with the old password die
   immediately - do this when traffic is low.

## 2. Supabase JWT secret (`SUPABASE_JWT_SECRET`)
Signs every user token; whoever holds it can sign in as anyone.
1. Supabase → Project Settings → JWT Keys → generate a new JWT secret.
   **This signs out every user.**
2. Set `SUPABASE_JWT_SECRET` on both services to the new value immediately
   after, so users can sign back in.
3. Also update any place that copied it (CI `ODE_QA_PASSWORD` tokens, local
   `.env` files).

## 3. Object storage (R2): `STORAGE_ACCESS_KEY_ID`, `STORAGE_SECRET_ACCESS_KEY`
1. Cloudflare → R2 → Manage API tokens → create a new token scoped to the
   bucket (Object Read & Write; add bucket admin so the API can set lifecycle
   rules, see LAUNCH_CHECKLIST N7).
2. Set both variables on both services, confirm uploads still work, then
   **delete the old token**.

## 4. Redis (`REDIS_PASSWORD`, Upstash)
1. Upstash console → your database → reset the password.
2. Set `REDIS_PASSWORD` (and `REDIS_ADDR` if the host changed) on both
   services. Queued jobs survive; if the database was recreated, entries stuck
   `pending` are re-queued by the reconciler within ~15 minutes.
3. Check the Upstash usage graph for traffic you don't recognise.

## 5. AI and speech providers
- `ANTHROPIC_API_KEY`: Anthropic Console → API keys → create new, set, delete old.
  Check Usage for unfamiliar spikes.
- `OPENAI_API_KEY`: OpenAI dashboard → API keys → same.
- `AZURE_TTS_KEY`: Azure portal → your Speech resource → Keys → Regenerate Key 1
  (and Key 2 once you have switched).

## 6. Firebase push (`FCM_CREDENTIALS_JSON`)
Whoever holds it can push to every user.
1. Google Cloud console → IAM → Service accounts →
   `firebase-adminsdk-fbsvc@dreamlog-48f94` → Keys → Add key → JSON.
2. Set `FCM_CREDENTIALS_JSON` to the **whole file contents** on both services.
3. Delete the old key (id starting `78270c9d`), keeping the system-managed ones.

## 7. Google OAuth (`WEB_CLIENT_SECRET`)
1. Google Cloud console → APIs & Services → Credentials → the Web client →
   reset secret.
2. Set `WEB_CLIENT_SECRET` on both services and update the Google provider
   secret in Supabase (Authentication → Providers → Google) if it uses the
   same client.

## 8. Notes encryption master key (`MASTER_ENCRYPTION_KEY`) - zero downtime
Each therapist's notes are encrypted with a data key that is wrapped by this
master key. Swapping the value alone would make every note unreadable, so
rotate with the previous-key fallback (needs a backend that includes
`MASTER_ENCRYPTION_KEY_PREVIOUS`, added 2026-10-23):
1. Generate a new key: `openssl rand -hex 32`
2. On **both** services set `MASTER_ENCRYPTION_KEY` = the new key and
   `MASTER_ENCRYPTION_KEY_PREVIOUS` = the old value. Deploy.
3. The API log shows `notes: master key re-wrap complete`. Notes stay readable
   throughout (anything still under the old key is opened with it and re-wrapped
   on use).
4. After that log line appears, remove `MASTER_ENCRYPTION_KEY_PREVIOUS` from
   both services. Keep the new key in your password manager - **losing it makes
   all therapist notes unrecoverable.**

Limit: this re-wraps the per-therapist data keys; the data keys themselves and
the stored note ciphertexts are unchanged. It protects against an attacker who
holds only a FUTURE database copy plus the old master key. An attacker who
already holds an old database dump AND the old master key can still read that
dump. Closing that needs rotating the data keys and re-encrypting every note
(a maintenance-window job); not implemented.

## 9. Housekeeping
- Rotate the demo account password (Supabase → Authentication → Users).
- `SENTRY_DSN`, `SUPABASE_URL`, `FCM_PROJECT_ID`, `WEB_CLIENT_ID` are not
  secrets and need no rotation.
- Revoke and recreate GitHub PATs / EAS tokens if they were ever stored in
  Railway.
- Afterwards: check API logs for the 401/500 rate, confirm a test sign-in, an
  upload, a therapy message and a push notification.
