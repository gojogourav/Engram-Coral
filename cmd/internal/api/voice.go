package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
)

type TranscribeRequest struct {
	Audio  string `json:"audio"`  // base64-encoded audio
	Locale string `json:"locale"` // hint: "hi", "en", "es" — Whisper will auto-detect if wrong
}

// TranscribeResponse returns both the raw transcript AND the detected language
// so the frontend passes the right locale to Lingo for translation.
type TranscribeResponse struct {
	Transcript string `json:"transcript"`
	Locale     string `json:"locale"` // ISO 639-1 code detected by Whisper
}

func localeToWhisper(locale string) string {
	mapping := map[string]string{
		"en": "en", "hi": "hi", "es": "es",
		"ja": "ja", "zh": "zh", "de": "de",
		"fr": "fr", "pt": "pt", "ko": "ko", "ar": "ar",
	}
	if lang, ok := mapping[locale]; ok {
		return lang
	}
	return "en"
}

func (g *Gateway) VoiceTranscribeHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "cannot read body", http.StatusBadRequest)
		return
	}

	var req TranscribeRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	if req.Audio == "" {
		http.Error(w, "audio field is required", http.StatusBadRequest)
		return
	}

	locale := req.Locale
	if locale == "" {
		locale = "en"
	}

	transcript, detectedLocale, err := g.transcribeWithGroq(req.Audio, locale)
	if err != nil {
		log.Printf("Groq Whisper error: %v", err)
		http.Error(w, fmt.Sprintf("transcription failed: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("🎙️ Transcript: %q | Locale hint: %s | Detected: %s", transcript, locale, detectedLocale)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(TranscribeResponse{
		Transcript: transcript,
		Locale:     detectedLocale,
	})
}

type groqTranscriptionResponse struct {
	Text     string `json:"text"`
	Language string `json:"language"` // Whisper returns detected language
	Error    *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (g *Gateway) transcribeWithGroq(audioBase64, locale string) (transcript, detectedLocale string, err error) {
	if g.GroqAPIKey == "" {
		return "", "", fmt.Errorf("GROQ_API_KEY not configured — free key at console.groq.com")
	}

	audioBytes, err := base64.StdEncoding.DecodeString(audioBase64)
	if err != nil {
		return "", "", fmt.Errorf("invalid base64 audio: %w", err)
	}

	// Groq rejects files under ~1KB — means recording was too short or empty
	if len(audioBytes) < 1024 {
		return "", "", fmt.Errorf("recording too short — hold the button for at least 1 second")
	}

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	//TODO: Understand this important part
	filePart, err := writer.CreateFormFile("file", "audio.m4a")
	if err != nil {
		return "", "", fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := filePart.Write(audioBytes); err != nil {
		return "", "", fmt.Errorf("failed to write audio: %w", err)
	}

	writer.WriteField("model", "whisper-large-v3")
	writer.WriteField("language", localeToWhisper(locale)) // hint, not enforced
	writer.WriteField("response_format", "verbose_json")   // gives us language detection
	writer.Close()

	httpReq, err := http.NewRequest("POST", "https://api.groq.com/openai/v1/audio/transcriptions", &buf)
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+g.GroqAPIKey)
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := g.HTTPClient.Do(httpReq)
	if err != nil {
		return "", "", fmt.Errorf("Groq request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("Groq error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var groqResp groqTranscriptionResponse
	if err := json.Unmarshal(respBody, &groqResp); err != nil {
		return "", "", fmt.Errorf("failed to decode response: %w", err)
	}

	if groqResp.Error != nil {
		return "", "", fmt.Errorf("Groq error: %s", groqResp.Error.Message)
	}

	if groqResp.Text == "" {
		return "", "", fmt.Errorf("empty transcript — audio may be too short or silent")
	}

	detected := groqResp.Language
	if detected == "" {
		detected = locale // fall back to user's hint
	}

	return groqResp.Text, detected, nil
}
