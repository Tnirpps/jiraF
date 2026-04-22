package tasklinks

import (
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func TestExtractFromText(t *testing.T) {
	links := ExtractFromText("логи https://logs.example.com/a, график https://grafana.example.com/d/1")

	if len(links) != 2 {
		t.Fatalf("expected 2 links, got %#v", links)
	}
	if links[0].URL != "https://logs.example.com/a" {
		t.Fatalf("expected first URL to trim punctuation, got %q", links[0].URL)
	}
}

func TestExtractFromTelegramMessageEntities(t *testing.T) {
	message := &tgbotapi.Message{
		Text: "docs and hidden",
		Entities: []tgbotapi.MessageEntity{
			{Type: "url", Offset: 0, Length: 4},
			{Type: "text_link", Offset: 9, Length: 6, URL: "https://hidden.example.com/doc"},
		},
	}

	links := ExtractFromTelegramMessage(message)

	if len(links) != 1 {
		t.Fatalf("expected only valid text_link, got %#v", links)
	}
	if links[0].URL != "https://hidden.example.com/doc" {
		t.Fatalf("unexpected text_link URL %q", links[0].URL)
	}
}

func TestExtractFromTelegramMessageDeduplicatesTextAndEntities(t *testing.T) {
	message := &tgbotapi.Message{
		Text: "https://docs.example.com/a",
		Entities: []tgbotapi.MessageEntity{
			{Type: "url", Offset: 0, Length: len("https://docs.example.com/a")},
		},
	}

	links := ExtractFromTelegramMessage(message)

	if len(links) != 1 {
		t.Fatalf("expected deduplicated link, got %#v", links)
	}
}

func TestNormalizeSelectedLinks(t *testing.T) {
	candidates := []LinkCandidate{
		{URL: "https://logs.example.com/a"},
		{URL: "https://docs.example.com/a"},
	}
	selected := []TaskLink{
		{URL: "https://logs.example.com/a", Role: "unknown", Reason: ""},
		{URL: "https://invented.example.com/a", Role: "docs", Reason: "лишняя ссылка"},
		{URL: "https://docs.example.com/a", Role: "docs", Reason: "документация"},
		{URL: "https://docs.example.com/a", Role: "docs", Reason: "дубль"},
	}

	result := NormalizeSelectedLinks(candidates, selected)

	if len(result) != 2 {
		t.Fatalf("expected 2 selected links, got %#v", result)
	}
	if result[0].Role != "other" {
		t.Fatalf("expected unknown role to normalize to other, got %q", result[0].Role)
	}
	if result[0].Reason == "" {
		t.Fatal("expected default reason for empty reason")
	}
	if result[1].URL != "https://docs.example.com/a" {
		t.Fatalf("expected docs link to remain, got %#v", result[1])
	}
}

func TestNormalizeSelectedLinksShortensLongReason(t *testing.T) {
	candidates := []LinkCandidate{{URL: "https://docs.example.com/a"}}
	selected := []TaskLink{{
		URL:    "https://docs.example.com/a",
		Role:   "docs",
		Reason: "эта ссылка содержит очень длинное подробное описание материала, которое не должно целиком попадать в сообщение Telegram и раздувать предпросмотр",
	}}

	result := NormalizeSelectedLinks(candidates, selected)

	if len(result) != 1 {
		t.Fatalf("expected 1 link, got %#v", result)
	}
	if len([]rune(result[0].Reason)) > 90 {
		t.Fatalf("expected reason to be shortened, got %d runes", len([]rune(result[0].Reason)))
	}
}
