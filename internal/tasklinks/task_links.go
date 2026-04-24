package tasklinks

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf16"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const MaxSelectedLinks = 10

var bareURLRe = regexp.MustCompile(`https?://[^\s<>()\[\]{}]+`)

type TaskLink struct {
	URL    string `json:"url"`
	Role   string `json:"role"`
	Reason string `json:"reason"`
}

type LinkCandidate struct {
	URL         string `json:"url"`
	MessageID   int    `json:"message_id,omitempty"`
	Username    string `json:"username,omitempty"`
	Timestamp   string `json:"timestamp,omitempty"`
	MessageText string `json:"message_text,omitempty"`
}

type TaskLinkSlice []TaskLink

func (s TaskLinkSlice) Value() (driver.Value, error) {
	if s == nil {
		return []byte("[]"), nil
	}

	data, err := json.Marshal([]TaskLink(s))
	if err != nil {
		return nil, fmt.Errorf("marshal task links: %w", err)
	}

	return data, nil
}

func (s *TaskLinkSlice) Scan(src any) error {
	if src == nil {
		*s = TaskLinkSlice{}
		return nil
	}

	var data []byte
	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("unsupported TaskLinkSlice source type %T", src)
	}

	if len(data) == 0 {
		*s = TaskLinkSlice{}
		return nil
	}

	var parsed []TaskLink
	if err := json.Unmarshal(data, &parsed); err != nil {
		return fmt.Errorf("unmarshal task links: %w", err)
	}

	*s = TaskLinkSlice(parsed)
	return nil
}

func ExtractFromTelegramMessage(message *tgbotapi.Message) []TaskLink {
	if message == nil {
		return nil
	}

	links := ExtractFromText(message.Text)
	for _, entity := range message.Entities {
		switch entity.Type {
		case "url":
			if extracted := extractEntityText(message.Text, entity.Offset, entity.Length); extracted != "" {
				links = append(links, TaskLink{URL: extracted})
			}
		case "text_link":
			if entity.URL != "" {
				links = append(links, TaskLink{URL: entity.URL})
			}
		}
	}

	return NormalizeLinks(links)
}

func ExtractFromText(text string) []TaskLink {
	matches := bareURLRe.FindAllString(text, -1)
	links := make([]TaskLink, 0, len(matches))
	for _, match := range matches {
		links = append(links, TaskLink{URL: match})
	}

	return NormalizeLinks(links)
}

func NormalizeLinks(links []TaskLink) []TaskLink {
	result := make([]TaskLink, 0, len(links))
	seen := make(map[string]struct{}, len(links))

	for _, link := range links {
		link.URL = normalizeURL(link.URL)
		if link.URL == "" {
			continue
		}

		key := strings.ToLower(link.URL)
		if _, ok := seen[key]; ok {
			continue
		}

		link.Role = NormalizeRole(link.Role)
		link.Reason = normalizeReason(link.Reason)
		seen[key] = struct{}{}
		result = append(result, link)
	}

	return result
}

func NormalizeSelectedLinks(candidates []LinkCandidate, selected []TaskLink) []TaskLink {
	allowed := make(map[string]string, len(candidates))
	for _, candidate := range candidates {
		normalized := normalizeURL(candidate.URL)
		if normalized == "" {
			continue
		}
		allowed[strings.ToLower(normalized)] = normalized
	}

	result := make([]TaskLink, 0, len(selected))
	seen := make(map[string]struct{}, len(selected))
	for _, link := range selected {
		normalized := normalizeURL(link.URL)
		if normalized == "" {
			continue
		}

		key := strings.ToLower(normalized)
		canonicalURL, ok := allowed[key]
		if !ok {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}

		seen[key] = struct{}{}
		result = append(result, TaskLink{
			URL:    canonicalURL,
			Role:   NormalizeRole(link.Role),
			Reason: normalizeReason(link.Reason),
		})
		if len(result) >= MaxSelectedLinks {
			break
		}
	}

	return result
}

func BuildCandidates(messages []MessageLike) []LinkCandidate {
	candidates := make([]LinkCandidate, 0)
	seen := make(map[string]struct{})

	for _, message := range messages {
		for _, link := range message.GetLinks() {
			normalized := normalizeURL(link.URL)
			if normalized == "" {
				continue
			}

			key := strings.ToLower(normalized)
			if _, ok := seen[key]; ok {
				continue
			}

			seen[key] = struct{}{}
			candidates = append(candidates, LinkCandidate{
				URL:         normalized,
				MessageID:   message.GetMessageID(),
				Username:    message.GetUsername(),
				Timestamp:   message.GetTimestamp().Format("2006-01-02 15:04:05"),
				MessageText: message.GetText(),
			})
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].URL < candidates[j].URL
	})

	return candidates
}

type MessageLike interface {
	GetLinks() []TaskLink
	GetMessageID() int
	GetUsername() string
	GetTimestamp() time.Time
	GetText() string
}

func NormalizeRole(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "logs", "metrics", "docs", "design", "chat":
		return strings.ToLower(strings.TrimSpace(role))
	default:
		return "other"
	}
}

func normalizeReason(reason string) string {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return "полезный материал для задачи"
	}

	runes := []rune(reason)
	if len(runes) > 90 {
		return string(runes[:90])
	}

	return reason
}

func normalizeURL(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimRight(raw, ".,;:!?)]}")
	if raw == "" {
		return ""
	}

	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}

	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
		return raw
	default:
		return ""
	}
}

func extractEntityText(text string, offset, length int) string {
	if text == "" || length <= 0 {
		return ""
	}

	utf16Text := utf16.Encode([]rune(text))
	if offset < 0 || offset >= len(utf16Text) {
		return ""
	}

	end := offset + length
	if end > len(utf16Text) {
		return ""
	}

	return string(utf16.Decode(utf16Text[offset:end]))
}
