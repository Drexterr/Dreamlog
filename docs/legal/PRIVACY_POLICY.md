# Privacy Policy

**DreamLog - Voice Journaling App**
**Effective Date:** 29 May 2026
**Last Updated:** 29 May 2026

---

## Plain-Language Summary

Before the legal detail: here is what actually happens to your data, in plain English.

- You speak. Your audio is uploaded to a secure server, transcribed, then **permanently deleted**. We never keep your audio.
- Your transcript is stored **encrypted** on our servers. We cannot read it without your account credentials.
- To generate your AI reflection, your transcript is sent to **Anthropic** (makers of Claude). Anthropic does not use your data to train their models and does not retain it after processing under our zero-retention agreement.
- Your mood scores, reflections, and journal history are stored encrypted on our servers and **auto-deleted after 180 days** unless you choose to keep them longer.
- We will never sell your data. We will never show you ads. We will never share your journal with your employer, insurer, or any third party - including in our B2B corporate wellness product.
- You can delete everything, permanently, at any time from Settings → Delete my account.

---

## 1. Who We Are

DreamLog ("we", "us", "our") is operated by [Company Name], registered in India. We provide a voice journaling application with AI-powered emotional reflections.

For privacy questions, contact us at: **privacy@dreamlog.app**

---

## 2. What Data We Collect

### 2.1 Data You Provide Directly

| Data | Purpose | Retained Until |
|---|---|---|
| Name and email address | Account creation and identification | Account deletion |
| Password (bcrypt hashed - we cannot reverse it) | Authentication | Account deletion |
| Audio recordings | Transcription only - deleted immediately after | **Deleted within minutes of upload** |
| Voice journal transcripts | AI analysis and your personal history | 180 days (configurable) or account deletion |
| Follow-up conversation messages | AI responses and your conversation history | 180 days or account deletion |
| User goal and preferences | Personalising AI reflections | Account deletion |
| Timezone and nudge preferences | Scheduling morning notifications | Account deletion |

### 2.2 Data Generated Automatically

| Data | Purpose | Retained Until |
|---|---|---|
| Mood scores and emotional tone analysis | Mood tracking and trend visualisation | 180 days or account deletion |
| AI-generated reflections and summaries | Display in app | 180 days or account deletion |
| FCM device tokens | Push notifications | Until you unregister the device or delete account |
| IP address and request logs | Security, abuse prevention | 30 days |
| App crash reports | Bug fixing | 30 days |

### 2.3 What We Do NOT Collect

- Precise GPS location
- Contacts or call history
- Other apps on your device
- Biometric data
- Financial information beyond subscription status

---

## 3. How We Use Your Data

We use your data **only** for the following purposes:

1. **Providing the service** - transcribing your recordings, generating AI reflections, storing your journal history
2. **Safety** - detecting crisis signals in transcripts and providing emergency resources (see Section 8)
3. **Personalisation** - using your past entries and stated goal to make reflections feel relevant to you
4. **Push notifications** - morning nudges you opt into; you can disable these at any time
5. **Customer support** - responding to requests you initiate
6. **Legal compliance** - meeting obligations under applicable law
7. **Security** - detecting and preventing unauthorised access

We do not use your data for advertising, profiling for third-party sale, or automated decisions that have legal or significant effects on you.

---

## 4. Third-Party Services

We share data with the following third parties, limited strictly to what is necessary:

### 4.1 Anthropic (AI Reflections)

Your journal transcript is sent to Anthropic's API to generate your reflection. We operate under Anthropic's **zero-data-retention** agreement, which means:

- Anthropic does not store your transcript after generating the response
- Anthropic does not use your data to train their AI models
- Data is transmitted over TLS and processed in Anthropic's secure infrastructure

**What Anthropic sees:** your transcript text (no name, no email, no account identifier)
**Anthropic's privacy policy:** [anthropic.com/privacy](https://www.anthropic.com/privacy)

### 4.2 OpenAI Whisper / Local Whisper (Transcription)

Audio is transcribed using Whisper. In production, this may use OpenAI's API or a self-hosted model depending on your region. If OpenAI's API is used, the audio is transmitted under their zero-retention API policy.

**What Whisper sees:** your raw audio recording (no account information)
**Audio is deleted immediately** after transcription regardless of which transcription method is used.

### 4.3 Cloudflare R2 (Temporary Audio Storage)

Audio is briefly stored on Cloudflare R2 while awaiting transcription. It is deleted within minutes of upload. Cloudflare R2 is SOC 2 compliant and data is encrypted at rest.

### 4.4 Firebase Cloud Messaging (Push Notifications)

Your FCM device token is shared with Google Firebase solely to deliver push notifications you have opted into. Firebase does not receive any journal content.

### 4.5 Payment Processors

Subscription payments are processed by [Razorpay / Stripe]. We do not store card numbers. Payment processors receive only your email and payment amount.

---

## 5. Data Storage and Security

### 5.1 Encryption

- **In transit:** All data is encrypted using TLS 1.3
- **At rest:** Journal transcripts, reflections, and conversation messages are encrypted using AES-256 in our database
- **Audio:** Never stored beyond the transcription window; deleted from storage servers immediately after processing

### 5.2 Data Retention

| Data Type | Default Retention | User Control |
|---|---|---|
| Audio recordings | Deleted within minutes | N/A |
| Transcripts and reflections | 180 days | Adjustable in Settings; or delete anytime |
| Mood and analysis data | 180 days | Adjustable in Settings |
| Account information | Until account deletion | Delete account in Settings |
| Security logs | 30 days | Not configurable (legal requirement) |

### 5.3 Where Data Is Stored

Your data is stored on servers located in [India / Singapore]. We do not transfer personal data outside of these jurisdictions except as described in Section 4 (Anthropic, OpenAI, Google Firebase - which operate under their own compliance frameworks).

### 5.4 Access Controls

- Our engineers do not have routine access to your journal content
- Database access requires multi-factor authentication and is logged
- We conduct periodic access reviews

---

## 6. Your Rights

Under the Digital Personal Data Protection Act 2023 (India) and, where applicable, GDPR, you have the right to:

| Right | How to Exercise |
|---|---|
| **Access** your data | Settings → Export my data |
| **Delete** your data | Settings → Delete my account (permanent, irreversible) |
| **Correct** inaccurate data | Settings → Edit profile |
| **Withdraw consent** | Delete your account at any time |
| **Data portability** | Settings → Export my data (JSON format) |
| **Complain** to a regulator | Contact the Data Protection Board of India or your local supervisory authority |

We will respond to verifiable data requests within **30 days**.

---

## 7. B2B Corporate Wellness - Special Provisions

If you use DreamLog through an employer's corporate wellness program:

- **Your employer cannot read your journal.** They receive only anonymised, aggregated mood statistics across teams of 5 or more people. Individual entries are never exposed.
- **You choose to participate.** Enrollment is voluntary and self-initiated.
- **Your employer cannot identify you** from any data they receive. The aggregation is designed to make individual identification impossible.
- **Leaving a company program** does not delete your account or journal data - it only removes you from the anonymised team aggregate.
- Crisis entries are **always excluded** from corporate dashboards.

---

## 8. Crisis Detection - How It Works and Its Limits

DreamLog uses automated crisis detection on your transcripts as a safety feature. This is not a substitute for emergency services.

**What it does:**
- Scans transcripts for phrases associated with crisis or self-harm
- If triggered, shows you mental health hotline information and support resources
- A flag is stored in your account to track that a crisis resource was shown

**What it does not do:**
- Contact anyone on your behalf
- Alert your employer, family, or any third party
- Guarantee detection of all crisis situations - it is a best-effort safety screen

**If you are in immediate danger:** call your local emergency services (India: **112**, mental health: **iCall 9152987821**, Vandrevala Foundation **1860-2662-345**).

Crisis entries are **excluded from all sharing features**, mood analytics shared with employers, and therapist dashboards unless you explicitly share them.

---

## 9. Therapist Dashboard - Special Provisions

If you link your account to a registered therapist on DreamLog:

- The therapist sees **AI-generated summaries** of your entries, not your raw transcripts
- Mood scores and topic trends are shared
- You can **revoke access** at any time from Settings → Linked Therapist
- Revocation is immediate and permanent

---

## 10. Children's Privacy

DreamLog is not intended for users under the age of 18. We do not knowingly collect personal data from minors. If you believe a minor has created an account, contact us at privacy@dreamlog.app and we will delete the account within 72 hours.

---

## 11. Cookies and Tracking

The mobile app does not use advertising cookies or cross-app trackers. We use no analytics SDKs that share data with third parties (e.g. no Facebook SDK, no Amplitude, no Mixpanel). Any analytics we collect is first-party and described in Section 2.2.

---

## 12. Changes to This Policy

We will notify you via in-app notification and email at least **30 days** before any material change to this policy. Continued use after that period constitutes acceptance of the updated policy.

Non-material changes (fixing typos, adding clarifications) may be made without notice but will be reflected in the "Last Updated" date above.

---

## 13. Contact

**Data Protection Officer:** [Name]
**Email:** privacy@dreamlog.app
**Address:** [Company Address], India

For urgent data protection concerns, we aim to respond within **48 hours**.
