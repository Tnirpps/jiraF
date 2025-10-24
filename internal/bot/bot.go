package bot

import (
	"fmt"
	"log"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/user/telegram-bot/internal/game"
)

type Bot struct {
	api         *tgbotapi.BotAPI
	gameManager *game.Manager
	wg          sync.WaitGroup
	stopCh      chan struct{}
}

func New(token string) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	gameManager := game.NewManager()

	return &Bot{
		api:         api,
		gameManager: gameManager,
		stopCh:      make(chan struct{}),
	}, nil
}

func (b *Bot) Start() error {
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60

	updates := b.api.GetUpdatesChan(updateConfig)

	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		b.handleUpdates(updates)
	}()

	return nil
}

func (b *Bot) Stop() {
	close(b.stopCh)
	b.api.StopReceivingUpdates()
	b.wg.Wait()
}

func (b *Bot) handleUpdates(updates tgbotapi.UpdatesChannel) {
	for {
		select {
		case <-b.stopCh:
			return
		case update, ok := <-updates:
			if !ok {
				return
			}
			b.handleUpdate(update)
		}
	}
}

func (b *Bot) handleUpdate(update tgbotapi.Update) {
	if update.Message != nil {
		b.handleMessage(update.Message)
		return
	}
}

func getIQDescription(iq int) string {
	switch {
	case iq < -100:
		return "Поздравляем! Вы официально амёба 🦠"
	case iq < -50:
		return "Вы эволюционировали до уровня губки 🧽"
	case iq < 0:
		return "Курица умнее вас, и это факт! 🐔"
	case iq < 50:
		return "Ваш интеллект на уровне домашнего картофеля 🥔"
	case iq < 100:
		return "Вы начинаете смутно понимать, что дважды два равно примерно четыре... 🧮"
	case iq < 150:
		return "Средний человек! Поздравляем с посредственностью! 🎉"
	case iq < 200:
		return "Вы уже мудрее среднестатистического политика! 🧠"
	case iq < 250:
		return "Да вы практически Эйнштейн местного масштаба! 👨‍🔬"
	case iq < 300:
		return "Илон Маск нервно курит в сторонке! 🚀"
	default:
		return "Искусственный интеллект скоро придёт за советом к вам! 🤖"
	}
}

func (b *Bot) handleMessage(message *tgbotapi.Message) {
	log.Printf("[%s] %s", message.From.UserName, message.Text)

	if !message.IsCommand() {
		// Send generic help message for non-commands
		msg := tgbotapi.NewMessage(message.Chat.ID, "Привет! Я бот, измеряющий ваш IQ. Используй /help чтобы увидеть список команд.")
		b.api.Send(msg)
		return
	}

	switch message.Command() {
	case "start":
		msg := tgbotapi.NewMessage(message.Chat.ID,
			"🧠 *Добро пожаловать в Тест IQ Bot!* 🧠\n\n"+
				"Я измеряю ваш интеллект по сверхточной научной методике!\n"+
				"Мои алгоритмы основаны на квантовой физике, нейронауке и сомнительной статистике!\n\n"+
				"Используйте /test чтобы проверить свой IQ.")
		msg.ParseMode = "Markdown"
		b.api.Send(msg)

	case "help":
		helpText := "*Команды IQ бота:*\n\n" +
			"/start - Начать великое интеллектуальное путешествие\n" +
			"/help - Показать эту сверхумную справку\n" +
			"/test - Пройти супернаучный тест IQ\n" +
			"/iq - Узнать свой текущий уровень интеллекта\n" +
			"/rating - Таблица гениев и... остальных"

		msg := tgbotapi.NewMessage(message.Chat.ID, helpText)
		msg.ParseMode = "Markdown"
		b.api.Send(msg)

	case "test", "play":
		points, totalIQ, allowed := b.gameManager.PlayGame(message.Chat.ID, message.From.UserName)

		if !allowed {
			msg := tgbotapi.NewMessage(message.Chat.ID,
				"⚠️ *Перегрев мозга!* ⚠️\n\n"+
					"Ваши нейроны устали! Пожалуйста, дайте им отдохнуть.\n"+
					"(Не более 10 тестов за 10 секунд)")
			msg.ParseMode = "Markdown"
			b.api.Send(msg)
			return
		}

		var resultText string
		if points > 0 {
			resultText = fmt.Sprintf("🎓 *Тест завершен!*\n\n"+
				"Вау! Ваш IQ вырос на *%d* пунктов!\n"+
				"Текущий IQ: *%d*\n\n%s",
				points, totalIQ, getIQDescription(totalIQ))
		} else if points == 0 {
			resultText = fmt.Sprintf("😐 *Тест завершен!*\n\n"+
				"Хм, ваш IQ не изменился. *±%d* пунктов.\n"+
				"Текущий IQ: *%d*\n\n%s",
				points, totalIQ, getIQDescription(totalIQ))
		} else {
			resultText = fmt.Sprintf("🤦‍♂️ *Тест завершен!*\n\n"+
				"Ой! Ваш IQ упал на *%d* пунктов!\n"+
				"Текущий IQ: *%d*\n\n%s",
				-points, totalIQ, getIQDescription(totalIQ))
		}

		msg := tgbotapi.NewMessage(message.Chat.ID, resultText)
		msg.ParseMode = "Markdown"
		b.api.Send(msg)

	case "iq", "score":
		iq := b.gameManager.GetUserScore(message.From.UserName)

		resultText := fmt.Sprintf("🧠 *Результат анализа мозга*\n\n"+
			"Ваш текущий IQ: *%d*\n\n%s",
			iq, getIQDescription(iq))

		msg := tgbotapi.NewMessage(message.Chat.ID, resultText)
		msg.ParseMode = "Markdown"
		b.api.Send(msg)

	case "rating", "top":
		leaderboard := b.gameManager.FormatLeaderboard()

		msg := tgbotapi.NewMessage(message.Chat.ID, "🏆 *Рейтинг интеллекта* 🏆\n\n"+leaderboard)
		msg.ParseMode = "Markdown"
		b.api.Send(msg)

	default:
		msg := tgbotapi.NewMessage(message.Chat.ID, "🤔 Эта команда слишком сложна для моего ИИ. Используйте /help для списка понятных команд.")
		b.api.Send(msg)
	}
}
