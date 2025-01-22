package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

type GameState struct {
	Word     string
	Attempts []string
	Language string
}

var (
	token        string
	userGames    = make(map[int64]*GameState)
	englishWords = []string{"apple", "grape", "mango", "berry", "lemon", "peach", "melon", "plums", "cherry", "guava"}
	russianWords = []string{"книга", "слово", "мирно", "игрок", "столб", "птица", "груша", "сахар", "яблок", "кадык"}
	maxAttempts  = 6
)

func loadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Ошибка загрузки .env файла: %v", err)
	}
	token = os.Getenv("TELEGRAM_BOT_TOKEN")
}

func main() {
	loadEnv()
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {
			chatID := update.Message.Chat.ID
			text := strings.ToLower(update.Message.Text)

			switch text {
			case "/start":
				startGame(chatID, bot)
			case "/play":
				startLanguageSelection(chatID, bot)
			default:
				handleGuess(chatID, text, bot)
			}
		} else if update.CallbackQuery != nil {
			handleLanguageSelection(update.CallbackQuery, bot)
		}
	}
}

func startGame(chatID int64, bot *tgbotapi.BotAPI) {
	msg := tgbotapi.NewMessage(chatID, `Добро пожаловать в игру Wordle!
Правила:
1. Угадайте слово, вводя пятибуквенные слова.
2. После каждой попытки бот покажет подсказку:
   - 🟩 — буква угадана и стоит на правильной позиции.
   - 🟨 — буква угадана, но стоит на неверной позиции.
   - ⬛ — буквы нет в слове.
3. Максимальное количество попыток: 6.
4. Если угадаете слово, бот поздравит вас. Если нет — покажет правильное слово.`)
	bot.Send(msg)

	startLanguageSelection(chatID, bot)
}

func startLanguageSelection(chatID int64, bot *tgbotapi.BotAPI) {
	buttons := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("ENG", "lang_eng"),
			tgbotapi.NewInlineKeyboardButtonData("РУС", "lang_rus"),
		),
	)
	msg := tgbotapi.NewMessage(chatID, "Выберите язык:")
	msg.ReplyMarkup = buttons
	bot.Send(msg)
}

func handleLanguageSelection(callback *tgbotapi.CallbackQuery, bot *tgbotapi.BotAPI) {
	chatID := callback.Message.Chat.ID
	var words []string
	var language string

	if callback.Data == "lang_eng" {
		words = englishWords
		language = "ENG"
	} else if callback.Data == "lang_rus" {
		words = russianWords
		language = "РУС"
	} else {
		return
	}

	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Вы выбрали язык: %s.", language))
	bot.Send(msg)

	startNewGame(chatID, words, bot, language)

	bot.Request(tgbotapi.CallbackConfig{
		CallbackQueryID: callback.ID,
		Text:            "Язык выбран",
		ShowAlert:       false,
	})
}

func startNewGame(chatID int64, words []string, bot *tgbotapi.BotAPI, language string) {
	rand.Seed(time.Now().UnixNano())
	word := words[rand.Intn(len(words))]

	userGames[chatID] = &GameState{
		Word:     word,
		Attempts: []string{},
		Language: language,
	}

	msg := tgbotapi.NewMessage(chatID, "Игра началась! Введите пятибуквенное слово.")
	bot.Send(msg)
}

func handleGuess(chatID int64, guess string, bot *tgbotapi.BotAPI) {
	game, exists := userGames[chatID]
	if !exists {
		msg := tgbotapi.NewMessage(chatID, "Игра не начата. Введите /play.")
		bot.Send(msg)
		return
	}

	if len([]rune(guess)) != 5 {
		msg := tgbotapi.NewMessage(chatID, "Введите слово из пяти букв.")
		bot.Send(msg)
		return
	}

	game.Attempts = append(game.Attempts, guess)

	// Создаем накопительное сообщение
	response := ""
	for _, attempt := range game.Attempts {
		response += generateFeedbackMessage(game.Word, attempt) + " \n"
	}

	if guess == game.Word {
		response = "Вы угадали слово! Поздравляем!\nХотите сыграть ещё раз? Нажмите /play"
		delete(userGames, chatID)
	} else if len(game.Attempts) >= maxAttempts {
		response = fmt.Sprintf("Вы проиграли. Загаданное слово: %s.\nХотите сыграть ещё раз? Нажмите /play", game.Word)
		delete(userGames, chatID)
	}

	msg := tgbotapi.NewMessage(chatID, response)
	bot.Send(msg)
}

func generateFeedbackMessage(word, guess string) string {
	var feedback []string
	wordRune := []rune(word)
	guessRune := []rune(guess)
	used := make([]bool, len(wordRune))

	for i := 0; i < len(wordRune); i++ {
		if i < len(guessRune) && guessRune[i] == wordRune[i] {
			feedback = append(feedback, fmt.Sprintf("%c 🟩", guessRune[i]))
			used[i] = true
		} else {
			feedback = append(feedback, "")
		}
	}

	for i := 0; i < len(wordRune); i++ {
		if feedback[i] == "" && i < len(guessRune) {
			found := false
			for j := 0; j < len(wordRune); j++ {
				if !used[j] && wordRune[j] == guessRune[i] {
					feedback[i] = fmt.Sprintf("%c 🟨", guessRune[i])
					used[j] = true
					found = true
					break
				}
			}
			if !found {
				feedback[i] = fmt.Sprintf("%c ⬛", guessRune[i])
			}
		}
	}

	return strings.Join(feedback, " ")
}
