/**
 * Country-aware crisis helplines.
 *
 * This is the single source of truth for the mobile crisis surfaces (reflection
 * crisis card, therapy crisis screen, Settings "Get help now" modal). It mirrors
 * the backend list in `backend/internal/services/crisis.go` (`countryHelplines`)
 * - keep the two in sync when adding or editing a country.
 *
 * The user's country is chosen at onboarding ("Where are you based?") and stored
 * on their profile as an ISO 3166-1 alpha-2 code. Resolve it with
 * `detectUserCountry()` from `./region`, then pass the code here. Unknown or
 * missing countries fall back to international resources.
 */

export type HelplineAction =
  | { kind: 'call'; value: string } // tel: digits, e.g. "988"
  | { kind: 'text'; value: string } // sms: short code, e.g. "741741"
  | { kind: 'link'; value: string }; // https URL

export type Helpline = {
  name: string;
  detail: string; // human label, e.g. "Call or text 988" / "9152987821"
  hours?: string; // e.g. "24/7", "Mon–Sat, 8 AM–10 PM"
  action: HelplineAction;
};

export type CountryHelplines = {
  code: string; // ISO alpha-2, or "INTL" for the fallback
  countryName: string;
  emergency?: string; // local emergency number, display only
  lines: Helpline[];
};

const call = (value: string): HelplineAction => ({ kind: 'call', value });
const text = (value: string): HelplineAction => ({ kind: 'text', value });
const link = (value: string): HelplineAction => ({ kind: 'link', value });

// Keyed by ISO 3166-1 alpha-2. Mirrors backend/internal/services/crisis.go.
const HELPLINES: Record<string, CountryHelplines> = {
  IN: {
    code: 'IN',
    countryName: 'India',
    emergency: '112',
    lines: [
      { name: 'iCall', detail: '9152987821', hours: 'Mon–Sat, 8 AM–10 PM', action: call('9152987821') },
      { name: 'Vandrevala Foundation', detail: '1860-2662-345', hours: '24/7', action: call('18602662345') },
      { name: 'Tele-MANAS (Govt. of India)', detail: '14416', hours: '24/7', action: call('14416') },
      { name: 'iCall Chat', detail: 'icallhelpline.org', action: link('https://icallhelpline.org') },
    ],
  },
  US: {
    code: 'US',
    countryName: 'United States',
    emergency: '911',
    lines: [
      { name: '988 Suicide & Crisis Lifeline', detail: 'Call or text 988', hours: '24/7', action: call('988') },
      { name: 'Crisis Text Line', detail: 'Text HOME to 741741', hours: '24/7', action: text('741741') },
      { name: 'Veterans Crisis Line', detail: '988, then press 1', hours: '24/7', action: call('988') },
    ],
  },
  GB: {
    code: 'GB',
    countryName: 'United Kingdom',
    emergency: '999',
    lines: [
      { name: 'Samaritans', detail: '116 123', hours: 'Free · 24/7', action: call('116123') },
      { name: 'Crisis Text Line', detail: 'Text SHOUT to 85258', hours: '24/7', action: text('85258') },
      { name: 'PAPYRUS (under 35)', detail: '0800 068 4141', action: call('08000684141') },
    ],
  },
  IE: {
    code: 'IE',
    countryName: 'Ireland',
    emergency: '112',
    lines: [
      { name: 'Samaritans', detail: '116 123', hours: 'Free · 24/7', action: call('116123') },
      { name: 'Pieta House', detail: '1800 247 247', hours: '24/7', action: call('1800247247') },
      { name: 'Crisis Text Line', detail: 'Text HELLO to 50808', action: text('50808') },
    ],
  },
  CA: {
    code: 'CA',
    countryName: 'Canada',
    emergency: '911',
    lines: [
      { name: 'Talk Suicide Canada', detail: '1-833-456-4566', hours: '24/7', action: call('18334564566') },
      { name: 'Kids Help Phone', detail: '1-800-668-6868', hours: '24/7', action: call('18006686868') },
      { name: 'Crisis Text Line', detail: 'Text HOME to 686868', action: text('686868') },
    ],
  },
  AU: {
    code: 'AU',
    countryName: 'Australia',
    emergency: '000',
    lines: [
      { name: 'Lifeline', detail: '13 11 14', hours: '24/7', action: call('131114') },
      { name: 'Beyond Blue', detail: '1300 22 4636', hours: '24/7', action: call('1300224636') },
      { name: 'Crisis Text Line', detail: 'Text HELLO to 741741', action: text('741741') },
    ],
  },
  NZ: {
    code: 'NZ',
    countryName: 'New Zealand',
    emergency: '111',
    lines: [
      { name: 'Lifeline Aotearoa', detail: '0800 543 354', hours: '24/7', action: call('0800543354') },
      { name: 'Suicide Crisis Helpline', detail: '0508 828 865', hours: '24/7', action: call('0508828865') },
      { name: 'Need to Talk?', detail: 'Call or text 1737', action: call('1737') },
    ],
  },
  DE: {
    code: 'DE',
    countryName: 'Germany',
    emergency: '112',
    lines: [
      { name: 'Telefonseelsorge', detail: '0800 111 0 111', hours: 'Free · 24/7', action: call('08001110111') },
      { name: 'Telefonseelsorge', detail: '0800 111 0 222', hours: 'Free · 24/7', action: call('08001110222') },
      { name: 'Online counselling', detail: 'online.telefonseelsorge.de', action: link('https://online.telefonseelsorge.de') },
    ],
  },
  FR: {
    code: 'FR',
    countryName: 'France',
    emergency: '112',
    lines: [
      { name: 'Prévention du Suicide', detail: '3114', hours: '24/7', action: call('3114') },
      { name: 'SOS Amitié', detail: '09 72 39 40 50', action: call('0972394050') },
    ],
  },
  NL: {
    code: 'NL',
    countryName: 'Netherlands',
    emergency: '112',
    lines: [
      { name: '113 Zelfmoordpreventie', detail: '0800-0113', hours: 'Free · 24/7', action: call('08000113') },
      { name: '113 Zelfmoordpreventie', detail: '113', hours: '24/7', action: call('113') },
    ],
  },
  BE: {
    code: 'BE',
    countryName: 'Belgium',
    emergency: '112',
    lines: [
      { name: 'Centre de Prévention du Suicide', detail: '0800 32 123', hours: '24/7', action: call('080032123') },
      { name: 'Zelfmoordlijn 1813', detail: '1813', hours: '24/7', action: call('1813') },
    ],
  },
  CH: {
    code: 'CH',
    countryName: 'Switzerland',
    emergency: '112',
    lines: [
      { name: 'Die Dargebotene Hand', detail: '143', hours: '24/7', action: call('143') },
      { name: 'Pro Juventute (under 25)', detail: '147', hours: '24/7', action: call('147') },
    ],
  },
  AT: {
    code: 'AT',
    countryName: 'Austria',
    emergency: '112',
    lines: [
      { name: 'Telefonseelsorge', detail: '142', hours: 'Free · 24/7', action: call('142') },
      { name: 'Rat auf Draht (youth)', detail: '147', hours: '24/7', action: call('147') },
    ],
  },
  ES: {
    code: 'ES',
    countryName: 'Spain',
    emergency: '112',
    lines: [
      { name: 'Línea de Atención a la Conducta Suicida', detail: '024', hours: '24/7', action: call('024') },
      { name: 'Teléfono de la Esperanza', detail: '717 003 717', hours: '24/7', action: call('717003717') },
    ],
  },
  IT: {
    code: 'IT',
    countryName: 'Italy',
    emergency: '112',
    lines: [
      { name: 'Telefono Amico Italia', detail: '02 2327 2327', action: call('0223272327') },
      { name: 'Samaritans Onlus', detail: '06 77208977', action: call('0677208977') },
    ],
  },
  PT: {
    code: 'PT',
    countryName: 'Portugal',
    emergency: '112',
    lines: [
      { name: 'SOS Voz Amiga', detail: '213 544 545', action: call('213544545') },
      { name: 'Conversa Amiga', detail: '808 237 327', action: call('808237327') },
    ],
  },
  SE: {
    code: 'SE',
    countryName: 'Sweden',
    emergency: '112',
    lines: [
      { name: 'Mind Självmordslinjen', detail: '90101', hours: '24/7', action: call('90101') },
      { name: 'Bris (under 18)', detail: '116 111', action: call('116111') },
    ],
  },
  NO: {
    code: 'NO',
    countryName: 'Norway',
    emergency: '113',
    lines: [
      { name: 'Mental Helse Hjelpetelefonen', detail: '116 123', hours: '24/7', action: call('116123') },
      { name: 'Kirkens SOS', detail: '22 40 00 40', hours: '24/7', action: call('22400040') },
    ],
  },
  DK: {
    code: 'DK',
    countryName: 'Denmark',
    emergency: '112',
    lines: [
      { name: 'Livslinien', detail: '70 201 201', action: call('70201201') },
    ],
  },
  SG: {
    code: 'SG',
    countryName: 'Singapore',
    emergency: '995',
    lines: [
      { name: 'Samaritans of Singapore (SOS)', detail: '1767', hours: '24/7', action: call('1767') },
      { name: 'IMH Crisis Helpline', detail: '6389 2222', hours: '24/7', action: call('63892222') },
    ],
  },
  PK: {
    code: 'PK',
    countryName: 'Pakistan',
    emergency: '115',
    lines: [
      { name: 'Umang', detail: '0317 4288665', action: call('03174288665') },
      { name: 'Rozan Counselling', detail: '0304 111 1741', action: call('03041111741') },
    ],
  },
  BD: {
    code: 'BD',
    countryName: 'Bangladesh',
    emergency: '999',
    lines: [
      { name: 'Kaan Pete Roi', detail: '09612 119 911', action: call('09612119911') },
      { name: 'Moner Bondhu', detail: '01776 632 344', action: call('01776632344') },
    ],
  },
  NG: {
    code: 'NG',
    countryName: 'Nigeria',
    emergency: '112',
    lines: [
      { name: 'Suicide Research & Prevention Initiative', detail: '0800 78774', action: call('080078774') },
      { name: 'Mentally Aware Nigeria', detail: 'mentallyaware.org', action: link('https://mentallyaware.org') },
    ],
  },
  ZA: {
    code: 'ZA',
    countryName: 'South Africa',
    emergency: '10111',
    lines: [
      { name: 'SADAG Suicide Crisis Line', detail: '0800 567 567', hours: '24/7', action: call('0800567567') },
      { name: 'Lifeline SA', detail: '0861 322 322', action: call('0861322322') },
    ],
  },
  BR: {
    code: 'BR',
    countryName: 'Brazil',
    emergency: '192',
    lines: [
      { name: 'CVV — Centro de Valorização da Vida', detail: '188', hours: '24/7', action: call('188') },
      { name: 'CVV Chat', detail: 'cvv.org.br', action: link('https://www.cvv.org.br') },
    ],
  },
  MX: {
    code: 'MX',
    countryName: 'Mexico',
    emergency: '911',
    lines: [
      { name: 'SAPTEL', detail: '55 5259 8121', hours: '24/7', action: call('5552598121') },
      { name: 'Línea de la Vida', detail: '800 911 2000', hours: '24/7', action: call('8009112000') },
    ],
  },
  JP: {
    code: 'JP',
    countryName: 'Japan',
    emergency: '119',
    lines: [
      { name: 'TELL Lifeline (English)', detail: '03-5774-0992', action: call('0357740992') },
      { name: 'Yorisoi Hotline', detail: '0120-279-338', hours: '24/7', action: call('0120279338') },
    ],
  },
  KR: {
    code: 'KR',
    countryName: 'South Korea',
    emergency: '119',
    lines: [
      { name: 'Suicide Prevention Hotline', detail: '1393', hours: '24/7', action: call('1393') },
      { name: 'LifeLine Korea', detail: '1588-9191', hours: '24/7', action: call('15889191') },
    ],
  },
  AE: {
    code: 'AE',
    countryName: 'United Arab Emirates',
    emergency: '999',
    lines: [
      { name: 'Estijaba (Dept. of Health)', detail: '800 1717', action: call('8001717') },
      { name: 'Mental Support Line', detail: '920 033 360', action: call('920033360') },
    ],
  },
};

// International fallback for any country not listed above.
const INTERNATIONAL: CountryHelplines = {
  code: 'INTL',
  countryName: 'International',
  lines: [
    { name: 'Find A Helpline', detail: 'findahelpline.com · 200+ countries', action: link('https://findahelpline.com/?utm_source=dreamlog') },
    { name: 'IASP Crisis Centres', detail: 'iasp.info', action: link('https://www.iasp.info/resources/Crisis_Centres/') },
  ],
};

/**
 * Returns crisis helplines for the given ISO 3166-1 alpha-2 country code.
 * Falls back to international resources when the code is empty or unknown
 * (including the onboarding "Other" choice).
 */
export function helplinesForCountry(code: string | undefined | null): CountryHelplines {
  const c = (code ?? '').toUpperCase();
  return HELPLINES[c] ?? INTERNATIONAL;
}

/** Builds the tap target URI for a helpline action. */
export function helplineHref(action: HelplineAction): string {
  switch (action.kind) {
    case 'call':
      return `tel:${action.value}`;
    case 'text':
      return `sms:${action.value}`;
    case 'link':
      return action.value;
  }
}
