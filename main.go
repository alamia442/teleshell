package main

import (
	"bytes"
	"os"
	"os/exec"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

const (
	CmdLogin  = "/login"
	CmdLogout = "/logout"
)

var (
	apiToken = os.Getenv("TELESHELL_API_TOKEN")
	password = os.Getenv("TELESHELL_PASSWORD")
	shell    = os.Getenv("TELESHELL_SHELL")
)

const (
	MaxMessageLength = 4096
	MaxChunkMessages = 10
)

const (
	ChatStateInitial = iota
	ChatStateAwaitingPassword
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
			default:
				// Switch chat state back to initial to rule out state traps.
				chats[update.Message.Chat.ID].State = ChatStateInitial

				if checkLogin(chats, update.Message, bot) {
					go func(message *tgbotapi.Message) {
						output, err := executeInShell(update.Message.Text)
						output = strings.Trim(output, "\n")

						// Prepare response message for command run.
						messageTextBuilder := strings.Builder{}

						offset0, length0 := messageTextBuilder.Len(), len("Output:")
						messageTextBuilder.WriteString("Output:\n")

						offset1, length1 := messageTextBuilder.Len(), len(output)
						messageTextBuilder.WriteString(output)

						if err != nil {
							// Prepare error response message for command run.
							messageTextBuilder.WriteString("\n\n")
							offset2, length2 := messageTextBuilder.Len(), len("Error:")
							messageTextBuilder.WriteString("Error:\n")

							offset3, length3 := messageTextBuilder.Len(), len(err.Error())
							messageTextBuilder.WriteString(err.Error())

							messageText := messageTextBuilder.String()
							messageConfig := newMessageConfig(message, messageText)
							messageConfig.Entities = append(messageConfig.Entities,
								tgbotapi.MessageEntity{Type: "bold", Offset: offset0, Length: length0},
								tgbotapi.MessageEntity{Type: "code", Offset: offset1, Length: length1},
								tgbotapi.MessageEntity{Type: "bold", Offset: offset2, Length: length2},
								tgbotapi.MessageEntity{Type: "code", Offset: offset3, Length: length3},
							)
							logSendMessage(bot.Send(messageConfig))
						} else {
							messageText := messageTextBuilder.String()
							messageConfig := newMessageConfig(message, messageText)
							messageConfig.Entities = append(messageConfig.Entities,
								tgbotapi.MessageEntity{Type: "bold", Offset: offset0, Length: length0},
								tgbotapi.MessageEntity{Type: "code", Offset: offset1, Length: length1},
							)
							logSendMessage(bot.Send(messageConfig))
						}
					}(update.Message)
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
	if err != nil {
		logEvent := log.Warn().Err(err)
		logEvent.Int("message-id", message.MessageID)
		logEvent.Str("message-text", message.Text)
		logEvent.Msg("Failed to send message")
		return
	}

	logEvent := log.Info()
	logEvent.Str("username", message.Chat.UserName)
	logEvent.Int("message-id", message.MessageID)
	logEvent.Str("message-text", message.Text)
	logEvent.Msg("Message sent")
}

// executeInShell executes specified script in Bash.
func executeInShell(script string) (string, error) {
	// Prepare command input.
	buffer := bytes.Buffer{}
	buffer.WriteString(script)

	// Prepare command instance.
	command := exec.Command(shell)
	command.Stdin = &buffer

	// Execute command capturing stdin and stdout.
	output, err := command.CombinedOutput()
	if err != nil {
		return string(output), errors.Wrap(err, "failed to execute command")
	}

	return string(output), nil
}
