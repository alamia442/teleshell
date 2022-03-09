package logging

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func init() {
	// Set output logging in colored text format.
	log.Logger = log.Output(zerolog.NewConsoleWriter(
		func(w *zerolog.ConsoleWriter) {
			w.TimeFormat = "[" + time.RFC822 + "]"
			w.FormatLevel = func(i interface{}) string {
				return strings.ToUpper(fmt.Sprintf("[%-5s]", i))
			}
		},
	))

	// Set logger for the tgbotapi library.
	logger := &LoggerAdapter{component: "tgbotapi"}
	if err := tgbotapi.SetLogger(logger); err != nil {
		log.Fatal().Err(err).Msg("Failed to set tgbotapi logger")
	}
}

// LoggerAdapter adapts zerolog logger for tgbotapi.
type LoggerAdapter struct {
	component string
}

// Println implements corresponding interface method.
func (l *LoggerAdapter) Println(v ...interface{}) {
	l.newEvent().Msg(strings.Trim(fmt.Sprint(v...), "\n"))
}

// Printf implements corresponding interface method.
func (l *LoggerAdapter) Printf(format string, v ...interface{}) {
	l.newEvent().Msg(strings.Trim(fmt.Sprintf(format, v...), "\n"))
}

// newEvent constricts new event.
func (l *LoggerAdapter) newEvent() *zerolog.Event {
	return log.Debug().Str("component", l.component)
}
