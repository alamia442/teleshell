package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	CmdLogin  = "/login"
	CmdLogout = "/logout"
	CmdShell  = "/shell"
)

var (
	apiToken = os.Getenv("TELESHELL_API_TOKEN")
	bashPath = os.Getenv("TELESHELL_BASH_PATH")
	password = os.Getenv("TELESHELL_PASSWORD")
)

const (
	ChatStateInitial = iota
	ChatStateAwaitingPassword
	ChatStateAwaitingCommand
)

type ChatState struct {
	State    int
	LoggedIn bool
}

func main() {
	// Initialize Telegram Bot API Client.
	bot, err := tgbotapi.NewBotAPI(apiToken)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create Bot API client")
	}

	// Bot API client successfully created and authenticated.
	log.Info().Str("username", bot.Self.UserName).Msgf("Authenticated in the Telegram API")

	// Set debug mode for the client.
	bot.Debug = true

	// Prepare updates configuration.
	update := tgbotapi.NewUpdate(0)
	update.Timeout = 60

	// Prepare updates channel.
	updates := bot.GetUpdatesChan(update)

	// Prepare chats state index.
	chats := map[int64]*ChatState{}

	// Handle all incoming events.
	for update := range updates {
		if update.Message != nil {
			logIncomingMessage(update.Message)

			// Save metadata of chat to the state.
			if _, ok := chats[update.Message.Chat.ID]; !ok {
				chats[update.Message.Chat.ID] = &ChatState{
					State: ChatStateInitial,
				}
			}

			// FSE for the chat.
			switch {

			// Handle login command.
			case update.Message.Text == CmdLogin:
				// Prepare response message to ask the user's password.
				messageConfig := newMessageConfig(update.Message, "Specify password")
				messageConfig.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true}
				logSendMessage(bot.Send(messageConfig))

				// Switch chat session state to awaiting password.
				chats[update.Message.Chat.ID].State = ChatStateAwaitingPassword

			// Handle login command password.
			case chats[update.Message.Chat.ID].State == ChatStateAwaitingPassword:
				// Switch chat state back to initial to rule out state traps.
				chats[update.Message.Chat.ID].State = ChatStateInitial

				if update.Message.Text != password {
					// Prepare response message for invalid password.
					messageConfig := newMessageConfig(update.Message, "Invalid password")
					logSendMessage(bot.Send(messageConfig))
				} else {
					// Prepare response message for valid password.
					messageConfig := newMessageConfig(update.Message, "Logged in")
					logSendMessage(bot.Send(messageConfig))

					// Switch chat session to authenticated.
					chats[update.Message.Chat.ID].LoggedIn = true
				}

			// Handle logout command.
			case update.Message.Text == CmdLogout:
				if checkLogin(chats, update.Message, bot) {
					// Prepare response message for successful logout.
					messageConfig := newMessageConfig(update.Message, "Logged out")
					logSendMessage(bot.Send(messageConfig))

					// Reset chat session state.
					chats[update.Message.Chat.ID].LoggedIn = false
				}

			// Handle shell command.
			case update.Message.Text == CmdShell:
				if checkLogin(chats, update.Message, bot) {
					// Prepare response message for command run.
					messageConfig := newMessageConfig(update.Message, "Specify command")
					messageConfig.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true}
					logSendMessage(bot.Send(messageConfig))

					// Switch chat session state to awaiting command.
					chats[update.Message.Chat.ID].State = ChatStateAwaitingCommand
				}

			// Continue handle shell command.
			case chats[update.Message.Chat.ID].State == ChatStateAwaitingCommand:
				// Switch chat state back to initial to rule out state traps.
				chats[update.Message.Chat.ID].State = ChatStateInitial

				if checkLogin(chats, update.Message, bot) {
					go func() {
						output, err := executeInBash(update.Message.Text)
						if err != nil {
							// Prepare error response message for command run.
							messageText := fmt.Sprintf("Output:\n%s\nError:\n%s", output, err)
							messageConfig := newMessageConfig(update.Message, messageText)
							logSendMessage(bot.Send(messageConfig))
						} else {
							// Prepare success response message for command run.
							messageText := fmt.Sprintf("Output:\n%s", output)
							messageConfig := newMessageConfig(update.Message, messageText)
							logSendMessage(bot.Send(messageConfig))
						}
					}()
				}

			default:
				if checkLogin(chats, update.Message, bot) {
					// Prepare response message for unknown command.
					messageConfig := newMessageConfig(update.Message, "Unknown command")
					logSendMessage(bot.Send(messageConfig))
				}
			}
		}
	}
}

// checkLogin checks that user in specified chat was logged in.
func checkLogin(chats map[int64]*ChatState, message *tgbotapi.Message, bot *tgbotapi.BotAPI) bool {
	if !chats[message.Chat.ID].LoggedIn {
		// Prepare response message for invalid authentication.
		messageConfig := newMessageConfig(message, "Not logged in")
		logSendMessage(bot.Send(messageConfig))
		return false
	}
	return true
}

// newMessageConfig returns new message prototype as a reply to another message.
func newMessageConfig(replyTo *tgbotapi.Message, messageText string) tgbotapi.MessageConfig {
	messageConfig := tgbotapi.NewMessage(replyTo.Chat.ID, messageText)
	messageConfig.ReplyToMessageID = replyTo.MessageID
	return messageConfig
}

// logIncomingMessage logs incoming message from the update object.
func logIncomingMessage(message *tgbotapi.Message) {
	logEvent := log.Info()
	logEvent.Str("username", message.From.UserName)
	logEvent.Int("message-id", message.MessageID)
	logEvent.Str("message-text", message.Text)
	logEvent.Msg("Message accepted")
}

// logSendMessage logs message.Send() invocation result.
func logSendMessage(message tgbotapi.Message, err error) {
	var logEvent *zerolog.Event
	if err != nil {
		logEvent = log.Warn()
	} else {
		logEvent = log.Info()
	}

	logEvent.Str("username", message.Chat.UserName)
	logEvent.Int("message-id", message.MessageID)
	logEvent.Str("message-text", message.Text)

	if err != nil {
		logEvent.Err(err).Msg("Failed to send message")
	} else {
		logEvent.Msg("Message sent")
	}
}

// executeInBash executes specified script in Bash.
func executeInBash(script string) (string, error) {
	// Prepare command input.
	buffer := bytes.Buffer{}
	buffer.WriteString(script)

	// Prepare command instance.
	command := exec.Command(bashPath)
	command.Stdin = &buffer

	// Execute command capturing stdin and stdout.
	output, err := command.CombinedOutput()
	if err != nil {
		return string(output), errors.Wrap(err, "failed to execute command")
	}

	return string(output), nil
}
