package services

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode"

	appconfig "github.com/dreamlog/backend/internal/config"
	"github.com/dreamlog/backend/internal/models"
)

// ── Voice mappings ───────────────────────────────────────────────────────────

// openAIPersonaVoice maps each therapy persona to its OpenAI TTS voice (ROADMAP 8b).
// Used only when Azure TTS is not configured.
var openAIPersonaVoice = map[models.TherapyPersona]string{
	models.PersonaComforting: "nova",
	models.PersonaRational:   "onyx",
	models.PersonaCBT:        "alloy",
	models.PersonaMindful:    "shimmer",
}

// azureVoiceEN maps personas to Azure neural voices for English sessions.
var azureVoiceEN = map[models.TherapyPersona]string{
	models.PersonaComforting: "en-US-JennyNeural",
	models.PersonaRational:   "en-US-BrandonNeural",
	models.PersonaCBT:        "en-US-AriaNeural",
	models.PersonaMindful:    "en-US-SaraNeural",
}

// azureStyleEN maps personas to mstts:express-as styles. Azure silently falls
// back to the voice's neutral style when a style isn't supported by that voice,
// so an imperfect mapping degrades gracefully rather than erroring.
var azureStyleEN = map[models.TherapyPersona]string{
	models.PersonaComforting: "empathetic",
	models.PersonaRational:   "calm",
	models.PersonaCBT:        "chat",
	models.PersonaMindful:    "gentle",
}

// azureVoiceHI maps personas to Hindi voices. Hindi neural voices don't support
// the express-as styles above, so no style map exists for them.
var azureVoiceHI = map[models.TherapyPersona]string{
	models.PersonaComforting: "hi-IN-SwaraNeural",
	models.PersonaRational:   "hi-IN-MadhurNeural",
	models.PersonaCBT:        "hi-IN-SwaraNeural",
	models.PersonaMindful:    "hi-IN-SwaraNeural",
}

// azureVoiceHD maps personas to DragonHD multilingual voices (AZURE_TTS_USE_HD=true).
// HD voices speak English, Hindi, and code-switched Hinglish with one voice and
// infer emotion from the text itself, so no express-as style is ever applied.
// All four are GA DragonHD voices. Used for English / foreign-language sessions.
var azureVoiceHD = map[models.TherapyPersona]string{
	models.PersonaComforting: "en-US-Jenny:DragonHDLatestNeural",   // warm female, parity with JennyNeural
	models.PersonaRational:   "en-US-Andrew2:DragonHDLatestNeural", // male, optimized for conversational content
	models.PersonaCBT:        "en-US-Aria:DragonHDLatestNeural",    // parity with AriaNeural
	models.PersonaMindful:    "en-US-Serena:DragonHDLatestNeural",  // calm, measured female
}

// azureVoiceHDIndian maps personas to Indian DragonHD voices, used in HD mode
// when the session language is Hindi (detected or user-preferred). Indian-accent
// multilingual voices handle Hindi, Indian English, and Hinglish naturally.
var azureVoiceHDIndian = map[models.TherapyPersona]string{
	models.PersonaComforting: "en-IN-Diya:DragonHDLatestNeural",  // warm female
	models.PersonaRational:   "en-IN-Arjun:DragonHDLatestNeural", // clear male
	models.PersonaCBT:        "en-IN-Diya:DragonHDLatestNeural",
	models.PersonaMindful:    "en-IN-Diya:DragonHDLatestNeural",
}

// azureVoiceByLanguage maps the remaining user-selectable voice languages
// (Settings → Voice language, validated against models.SupportedVoiceLanguages)
// to one standard neural voice per language. English and Hindi keep their
// richer per-persona maps above; for these languages all personas share a
// single warm conversational voice and differ only by system prompt.
var azureVoiceByLanguage = map[string]string{
	"arabic":     "ar-SA-ZariyahNeural",
	"bengali":    "bn-IN-TanishaaNeural",
	"chinese":    "zh-CN-XiaoxiaoNeural",
	"dutch":      "nl-NL-ColetteNeural",
	"french":     "fr-FR-DeniseNeural",
	"german":     "de-DE-KatjaNeural",
	"greek":      "el-GR-AthinaNeural",
	"gujarati":   "gu-IN-DhwaniNeural",
	"indonesian": "id-ID-GadisNeural",
	"italian":    "it-IT-ElsaNeural",
	"japanese":   "ja-JP-NanamiNeural",
	"kannada":    "kn-IN-SapnaNeural",
	"korean":     "ko-KR-SunHiNeural",
	"malayalam":  "ml-IN-SobhanaNeural",
	"marathi":    "mr-IN-AarohiNeural",
	"polish":     "pl-PL-ZofiaNeural",
	"portuguese": "pt-BR-FranciscaNeural",
	"punjabi":    "pa-IN-VaaniNeural",
	"russian":    "ru-RU-SvetlanaNeural",
	"spanish":    "es-ES-ElviraNeural",
	"swedish":    "sv-SE-SofieNeural",
	"tamil":      "ta-IN-PallaviNeural",
	"telugu":     "te-IN-ShrutiNeural",
	"thai":       "th-TH-PremwadeeNeural",
	"turkish":    "tr-TR-EmelNeural",
	"ukrainian":  "uk-UA-PolinaNeural",
	"urdu":       "ur-IN-GulNeural",
	"vietnamese": "vi-VN-HoaiMyNeural",
}

type ttsStorageClient interface {
	Upload(ctx context.Context, key, contentType string, body io.Reader) error
	PresignDownload(ctx context.Context, key string, expiry time.Duration) (string, error)
}

// TTSService generates AI voice audio and stores it for presigned access.
// Provider selection: Azure Speech when AZURE_TTS_KEY+REGION are set (empathetic
// SSML styles + Hindi voices), otherwise OpenAI TTS when OPENAI_API_KEY is set,
// otherwise all calls are no-ops (returns empty string, nil error - dev mode).
type TTSService struct {
	openAICfg *appconfig.OpenAIConfig
	azureCfg  *appconfig.AzureTTSConfig
	storage   ttsStorageClient
	client    *http.Client
}

func NewTTSService(openAICfg *appconfig.OpenAIConfig, azureCfg *appconfig.AzureTTSConfig, storage ttsStorageClient) *TTSService {
	return &TTSService{
		openAICfg: openAICfg,
		azureCfg:  azureCfg,
		storage:   storage,
		client:    &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *TTSService) azureEnabled() bool {
	return s.azureCfg != nil && s.azureCfg.Key != "" && (s.azureCfg.Region != "" || s.azureCfg.BaseURL != "")
}

// Synthesize converts text to speech for the given persona and session language,
// uploads it to storage, and returns a short-lived presigned GET URL.
// language is the Whisper-detected language of the user's speech ("english",
// "hindi", "hi", ...); empty means unknown and defaults to English.
// Returns ("", nil) when TTS is disabled.
func (s *TTSService) Synthesize(ctx context.Context, sessionID, messageID, text string, persona models.TherapyPersona, language string) (string, error) {
	var (
		audioBytes []byte
		err        error
	)

	switch {
	case s.azureEnabled():
		audioBytes, err = s.callAzureTTS(ctx, text, persona, language)
	case s.openAICfg != nil && s.openAICfg.APIKey != "":
		audioBytes, err = s.callOpenAITTS(ctx, text, openAIVoiceFor(persona))
	default:
		return "", nil // TTS disabled (dev)
	}
	if err != nil {
		return "", fmt.Errorf("tts: synthesize: %w", err)
	}

	key := fmt.Sprintf("tts/%s/%s.mp3", sessionID, messageID)
	if err := s.storage.Upload(ctx, key, "audio/mpeg", bytes.NewReader(audioBytes)); err != nil {
		return "", fmt.Errorf("tts: upload: %w", err)
	}

	url, err := s.storage.PresignDownload(ctx, key, 5*time.Minute)
	if err != nil {
		return "", fmt.Errorf("tts: presign: %w", err)
	}

	return url, nil
}

// ── Azure Speech ─────────────────────────────────────────────────────────────

// selectAzureVoice picks the voice + express-as style for a persona and language.
// Precedence: VoiceOverride (one voice for everything) > UseHD (per-persona
// multilingual DragonHD voices) > per-language standard neural maps. HD voices
// detect emotion from the text itself, so they never get an express-as style.
func (s *TTSService) selectAzureVoice(persona models.TherapyPersona, language string) (voice, style string) {
	if s.azureCfg.VoiceOverride != "" {
		voice = s.azureCfg.VoiceOverride
	} else if s.azureCfg.UseHD {
		hdMap := azureVoiceHD
		if isHindiLanguage(language) {
			hdMap = azureVoiceHDIndian
		}
		voice = hdMap[persona]
		if voice == "" {
			voice = hdMap[models.PersonaComforting]
		}
	} else if isHindiLanguage(language) {
		voice = azureVoiceHI[persona]
		if voice == "" {
			voice = azureVoiceHI[models.PersonaComforting]
		}
	} else if v, ok := azureVoiceByLanguage[normalizeLanguage(language)]; ok {
		voice = v
	} else {
		voice = azureVoiceEN[persona]
		if voice == "" {
			voice = azureVoiceEN[models.PersonaComforting]
		}
		style = azureStyleEN[persona]
	}

	// DragonHD voices reject mstts:express-as (they infer emotion from content);
	// only standard neural voices get a style wrapper.
	if isAzureHDVoice(voice) {
		style = ""
	}
	return voice, style
}

// isAzureHDVoice reports whether the voice name refers to an HD base model
// (e.g. "en-IN-Aarti:DragonHDLatestNeural", "en-us-ava:DragonHDOmniLatestNeural").
func isAzureHDVoice(voice string) bool {
	return strings.Contains(strings.ToLower(voice), ":dragonhd")
}

// isHindiLanguage normalises Whisper's language output ("hindi", "hi", "Hindi").
func isHindiLanguage(language string) bool {
	switch normalizeLanguage(language) {
	case "hi", "hin", "hindi":
		return true
	}
	return false
}

// normalizeLanguage lower-cases and trims a Whisper-detected or user-selected
// language so it can be matched against the voice maps.
func normalizeLanguage(language string) string {
	return strings.ToLower(strings.TrimSpace(language))
}

// DetectTextLanguage gives a best-effort language for typed (non-voice) turns
// where Whisper never ran: any Devanagari codepoint marks the text as Hindi.
// Romanised Hinglish stays "english" - the multilingual voices handle it.
func DetectTextLanguage(text string) string {
	for _, r := range text {
		if unicode.Is(unicode.Devanagari, r) {
			return "hindi"
		}
	}
	return "english"
}

// buildAzureSSML renders the SSML document. Text is XML-escaped; the express-as
// wrapper is included only when a style is set (standard neural voices only).
func buildAzureSSML(text, voice, style string) string {
	var escaped bytes.Buffer
	_ = xml.EscapeText(&escaped, []byte(text))

	inner := escaped.String()
	if style != "" {
		inner = fmt.Sprintf(`<mstts:express-as style="%s">%s</mstts:express-as>`, style, inner)
	}

	lang := "en-US"
	if parts := strings.SplitN(voice, "-", 3); len(parts) == 3 {
		lang = parts[0] + "-" + strings.ToUpper(parts[1])
	}

	return fmt.Sprintf(
		`<speak version='1.0' xmlns='http://www.w3.org/2001/10/synthesis' xmlns:mstts='https://www.w3.org/2001/mstts' xml:lang='%s'><voice name='%s'>%s</voice></speak>`,
		lang, voice, inner,
	)
}

func (s *TTSService) azureEndpoint() string {
	if s.azureCfg.BaseURL != "" {
		return strings.TrimRight(s.azureCfg.BaseURL, "/") + "/cognitiveservices/v1"
	}
	return fmt.Sprintf("https://%s.tts.speech.microsoft.com/cognitiveservices/v1", s.azureCfg.Region)
}

func (s *TTSService) callAzureTTS(ctx context.Context, text string, persona models.TherapyPersona, language string) ([]byte, error) {
	voice, style := s.selectAzureVoice(persona, language)
	ssml := buildAzureSSML(text, voice, style)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.azureEndpoint(), strings.NewReader(ssml))
	if err != nil {
		return nil, fmt.Errorf("azure new request: %w", err)
	}
	req.Header.Set("Ocp-Apim-Subscription-Key", s.azureCfg.Key)
	req.Header.Set("Content-Type", "application/ssml+xml")
	req.Header.Set("X-Microsoft-OutputFormat", "audio-24khz-96kbitrate-mono-mp3")
	req.Header.Set("User-Agent", "dreamlog-backend")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("azure http do: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("azure read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("azure API returned %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// ── OpenAI TTS (fallback) ────────────────────────────────────────────────────

func openAIVoiceFor(persona models.TherapyPersona) string {
	if v := openAIPersonaVoice[persona]; v != "" {
		return v
	}
	return "nova"
}

func (s *TTSService) callOpenAITTS(ctx context.Context, text, voice string) ([]byte, error) {
	payload, err := json.Marshal(map[string]string{
		"model": "tts-1",
		"input": text,
		"voice": voice,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	url := s.openAICfg.BaseURL + "/audio/speech"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.openAICfg.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}
