package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	tg "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	botToken = "Token_bot"
	chatID   = 123456789 //your id
)

var bot *tg.BotAPI

func cleanText(s string) string {
	s = strings.TrimSpace(s)
	re := regexp.MustCompile(`\s+`)
	return re.ReplaceAllString(s, " ")
}

func getMatches() map[string]string {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", "https://soccer365.ru/online/&tab=1", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	matches := make(map[string]string)

	doc.Find(".game_block.online").Each(func(i int, s *goquery.Selection) {
		id, exists := s.Attr("id")
		if !exists {
			return
		}

		home := cleanText(s.Find(".ht .name span").Text())
		away := cleanText(s.Find(".at .name span").Text())
		homeScore := cleanText(s.Find(".ht .gls").Text())
		awayScore := cleanText(s.Find(".at .gls").Text())

		text := fmt.Sprintf("%s %s - %s %s", home, homeScore, awayScore, away)
		matches[id] = text
	})

	return matches
}

func logToCSV(filename, event, match string) {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println("Ошибка при открытии CSV:", err)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	err = writer.Write([]string{timestamp, event, match})
	if err != nil {
		log.Println("Ошибка при записи в CSV:", err)
	}
}

func sendToTelegram(message string) {
	if bot == nil {
		var err error
		bot, err = tg.NewBotAPI(botToken)
		if err != nil {
			log.Println("Ошибка инициализации бота:", err)
			return
		}
	}

	msg := tg.NewMessage(chatID, message)
	_, err := bot.Send(msg)
	if err != nil {
		log.Println("Ошибка отправки в Telegram:", err)
	}
}

func main() {
	last_matches := make(map[string]string)
	const logFile = "matches_log.csv"
	var idleCount int

	fmt.Println("Запуск трекера матчей...")
	sendToTelegram("📡 Бот запущен. Трекер начал отслеживание матчей.")

	for {
		current_matches := getMatches()

		if len(last_matches) == 0 {
			fmt.Println("Первый запуск, сохраняем текущее состояние...")
			for _, match := range current_matches {
				logToCSV(logFile, "Первый запуск", match)
			}
			last_matches = current_matches
		} else {
			for id, match := range current_matches {
				oldMatch, exists := last_matches[id]
				if !exists {
					fmt.Println("⚽ Новый матч:", match)
					logToCSV(logFile, "Новый матч", match)
					sendToTelegram("⚽ Новый матч: " + match)
				} else if oldMatch != match {
					fmt.Println("🔄 Обновление:", match)
					logToCSV(logFile, "Обновление", match)
					sendToTelegram("🔄 Обновление: " + match)
				}
			}

			for id, match := range last_matches {
				if _, exists := current_matches[id]; !exists {
					fmt.Println("🏁 Матч завершён:", match)
					logToCSV(logFile, "Матч завершён", match)
					sendToTelegram("🏁 Матч завершён: " + match)
				}
			}

			last_matches = current_matches
			idleCount++
			fmt.Printf("Цикл #%d завершён. Ожидание...\n", idleCount)
		}

		time.Sleep(60 * time.Second)
	}
}
