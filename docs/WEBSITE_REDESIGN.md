# Approved website redesign — 20 October 2026

The approved homepage and supporting-page previews are implemented in the existing
Next.js/Firebase website, including the therapist workspace. Backend contracts and
mobile code are unchanged.

## Structure

- `src/content/landing-markup.json`: reviewed homepage markup; static, trusted content.
- `src/components/landing-interactions.js`: approved ScrollCraft runtime and goal/
  screenshot dialogs, mounted and disposed with the React page.
- `src/content/public-pages.json`: reviewed public copy and the existing legal text.
  Never insert API responses or user data into these HTML templates.
- `SiteFrame`, `PublicPage`, `PortalShell`: public and authenticated page framing.
- `src/styles/`: scoped website and homepage styling; locally hosted brand fonts.
- Existing `/login`, `/dashboard`, client brief and session-note routes call the
  existing typed Axios client. No preview passwords, fabricated clients or mock
  authentication are included in the deployed app.

Privacy and Terms keep all 26 existing sections and their existing dates. This
change does not revise the policy text. The Teams page omits unsupported statistics
and regulatory assertions and retains the existing pricing and pilot enquiry.
Launch actions use email while store listings are unavailable. All existing public
destinations remain discoverable from the footer.

## Portal behavior

The backend client summary supplies an average, not a longitudinal trend. The
overview therefore labels that column **30-day mood**, without inferring a trend
from the absolute score. The individual brief displays the actual server trend.

Linking is a pending request: the client must approve it in the mobile app before
access appears. Consent is unlocked only after the consent endpoint succeeds.
Failed saves preserve unsaved text; failed deletions preserve session records.
Session switching and leaving the page protect unsaved edits. Images retain the
existing presign → PUT → create-session → poll OCR pipeline; notes and summaries
are real API operations, not preview simulations.

Firebase rewrites direct `/dashboard/clients/**` loads to the exported client-brief
shell. The client ID is read from the actual browser URL, not the static `_`
placeholder. Unknown/unshared IDs remain subject to backend authorization.

## Verification

- Production Next.js export with TypeScript checking.
- `node scripts/check-website-export.cjs`: exported route, fragment and asset checks;
  legal section counts; absence of review banners and preview credentials.
- Start `postman simulation run postman/simulations/portal.sim.yaml --port 4920`,
  then `node scripts/test-portal-ui.cjs`: actual React components and Axios client
  over real loopback HTTP with fictional data. Covers login, clients, pending links,
  consent failure/success, note save failures, summaries, deletion failures and
  direct client brief paths. Requires the existing mobile/portal npm dependencies.
- Browser checks cover public pages, real portal screens against that simulation,
  goal selection, pricing, screenshot enlargement, and mobile widths 320/390px.

The local mock is not a production backend test. Live checks avoid real client
records, journal submissions, paid AI calls and account creation. Firebase Hosting
deployment does not deploy backend services or publish a mobile OTA.

### Responsive follow-up — 20 October 2026

The homepage, therapist information, About, Teams, pricing, privacy, terms, login,
client overview and session notes were checked at 320×700, 390×844, 600×800,
768×1024, 1024×600, 1440×900, 2560×1440 and 844×390. Checks compare document
width and visible content bounds with the usable viewport, alongside visual
inspection of phone layouts and the goal dialog. The exported client brief is
checked with fictional local API data because Next's development server does not
apply Firebase's dynamic-client rewrite.

Reopening the goal chooser now centres it vertically on short screens so its
close button stays reachable; its contents scroll inside the viewport. The hero
logo reveals through a left-to-right SVG clipping rectangle, avoiding normalized
dash lengths that previously left a gap. Its final state retains the complete
line and dot. The pre-animation state prevents a flash of the completed logo;
reduced-motion users receive the complete static mark.
The floating goal action appears after its header equivalent leaves view, keeping
the hero's launch information clear. Its observer disconnects when leaving the
homepage.

## Deployment

### Journal journey and launch waitlist (26 October 2026)

The existing ScrollCraft three-beat journal scene is pinned on roomy desktop
viewports; compact and reduced-motion layouts keep all three beats readable in
ordinary order. A late CSS override had previously flattened the desktop scene.

The hero and final launch actions open one goal-themed waitlist dialog. It asks
only for email and states the launch-email purpose. `POST /waitlist` writes the
normalized email to `launch_waitlist` in the existing Supabase PostgreSQL database.
The goal stays in browser storage. The table has RLS enabled, no anonymous or
authenticated role grant, no public list endpoint, and an API-only unique key.
Duplicates return the same confirmation. The API uses a per-IP limiter and an
unadvertised honeypot. Migration 000049 is applied at API startup after backend
deployment. The privacy page explains collection, retention, and removal by email.
Test with `go test ./internal/handlers ./internal/services ./internal/repositories`
and the isolated integration database; the local Postman simulation exercises
the real React submit path and failure recovery. No launch emails are sent by
this change; sending the notice is an operational step at store launch.


From `therapist-portal`, build with
`NEXT_PUBLIC_API_URL=https://dreamlog-production-f9e2.up.railway.app` and run
`firebase deploy --only hosting --project dreamlog-48f94` after checks pass.
Never deploy a build pointed at the local simulation. Font assets and screenshots
are hosted locally under `public/`; existing CSP and security headers are retained.
