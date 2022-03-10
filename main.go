package main

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

const (
	// CmdLogin specifies chat login command.
	CmdLogin = "/login"

	// CmdLogout specifies chat logout command.
	CmdLogout = "/logout"

	// CmdDisplay specifies file display command.
	CmdDisplay = "/display"

	// CmdUpload specifies upload file command.
	CmdUpload = "/upload"
)

var (
	apiToken = os.Getenv("TELESHELL_API_TOKEN")
	password = os.Getenv("TELESHELL_PASSWORD")
	shell    = os.Getenv("TELESHELL_SHELL")
)

const (
	// ChatStateInitial represents initial chat state.
	ChatStateInitial = iota

	// ChatStateAwaitingPassword represents awaiting password state.
	ChatStateAwaitingPassword

	// ChatStateAwaitingDisplayPath represents awaiting display path state.
	ChatStateAwaitingDisplayPath

	// ChatStateAwaitingUploadPath represents awaiting upload path state.
	ChatStateAwaitingUploadPath

	// ChatStateAwaitingUploadFile represents awaiting upload file state.
	ChatStateAwaitingUploadFile
)

// ChatState represents chat state.
type ChatState struct {
	State      int
	LoggedIn   bool
	UploadPath string
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

			// Handle login with args command.
			case strings.HasPrefix(update.Message.Text, CmdLogin):
				commandArgs := strings.TrimPrefix(update.Message.Text, CmdLogin)
				update.Message.Text = strings.Trim(commandArgs, " ")
				fallthrough

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

			// Handle display command.
			case update.Message.Text == CmdDisplay:
				if checkLogin(chats, update.Message, bot) {
					// Prepare response message for display file command.
					messageConfig := newMessageConfig(update.Message, "Specify path")
					logSendMessage(bot.Send(messageConfig))

					// Switch chat session state to awaiting display path.
					chats[update.Message.Chat.ID].State = ChatStateAwaitingDisplayPath
				}

			// Handle display with args command.
			case strings.HasPrefix(update.Message.Text, CmdDisplay):
				commandArgs := strings.TrimPrefix(update.Message.Text, CmdDisplay)
				update.Message.Text = strings.Trim(commandArgs, " ")
				fallthrough

			// Handle display path command.
			case chats[update.Message.Chat.ID].State == ChatStateAwaitingDisplayPath:
				// Switch chat state back to initial to rule out state traps.
				chats[update.Message.Chat.ID].State = ChatStateInitial

				if checkLogin(chats, update.Message, bot) {
					fileBytes, err := getFileBytes(update.Message.Text)
					if err != nil {
						// Prepare response message for error.
						messageConfig := newMessageConfig(update.Message, err.Error())
						logSendMessage(bot.Send(messageConfig))
					} else {
						// Prepare response message with a file.
						messageConfig := tgbotapi.NewDocument(update.Message.Chat.ID, fileBytes)
						messageConfig.ReplyToMessageID = update.Message.MessageID
						logSendMessage(bot.Send(messageConfig))
					}
				}

			// Handle upload command.
			case update.Message.Text == CmdUpload:
				if checkLogin(chats, update.Message, bot) {
					// Prepare response message for upload file command.
					messageConfig := newMessageConfig(update.Message, "Specify upload path")
					logSendMessage(bot.Send(messageConfig))

					// Switch chat session state to awaiting upload path.
					chats[update.Message.Chat.ID].State = ChatStateAwaitingUploadPath
				}

			// Handle upload with args command.
			case strings.HasPrefix(update.Message.Text, CmdUpload):
				commandArgs := strings.TrimPrefix(update.Message.Text, CmdUpload)
				update.Message.Text = strings.Trim(commandArgs, " ")
				fallthrough

			// Handle upload path command.
			case chats[update.Message.Chat.ID].State == ChatStateAwaitingUploadPath:
				// Switch chat state back to initial to rule out state traps.
				chats[update.Message.Chat.ID].State = ChatStateInitial

				if checkLogin(chats, update.Message, bot) {
					chats[update.Message.Chat.ID].State = ChatStateAwaitingUploadFile
					chats[update.Message.Chat.ID].UploadPath = update.Message.Text

					// Prepare response message with a file path request.
					messageConfig := newMessageConfig(update.Message, "Specify file attachment")
					logSendMessage(bot.Send(messageConfig))
				}

			// Handle upload file command.
			case chats[update.Message.Chat.ID].State == ChatStateAwaitingUploadFile:
				// Switch chat state back to initial to rule out state traps.
				chats[update.Message.Chat.ID].State = ChatStateInitial

				if checkLogin(chats, update.Message, bot) {
					if update.Message.Document == nil {
						// Prepare response message for error.
						messageConfig := newMessageConfig(update.Message, "No file uploaded")
						logSendMessage(bot.Send(messageConfig))
					} else {
						go func(message *tgbotapi.Message, uploadPath string) {
							// Get URL for attachment file.
							fileURL, err := bot.GetFileDirectURL(update.Message.Document.FileID)
							if err != nil {
								// Prepare response message for error.
								err = errors.Wrap(err, "failed to get file URL")
								messageConfig := newMessageConfig(update.Message, err.Error())
								logSendMessage(bot.Send(messageConfig))
								return
							}

							// Download attachment file.
							fileResponse, err := http.Get(fileURL)
							if err != nil {
								// Prepare response message for error.
								err = errors.Wrap(err, "failed to download file")
								messageConfig := newMessageConfig(update.Message, err.Error())
								logSendMessage(bot.Send(messageConfig))
								return
							}
							defer func() {
								_ = fileResponse.Body.Close()
							}()

							// Read attachment file content.
							fileData, err := ioutil.ReadAll(fileResponse.Body)
							if err != nil {
								// Prepare response message for error.
								err = errors.Wrap(err, "failed to read file content")
								messageConfig := newMessageConfig(update.Message, err.Error())
								logSendMessage(bot.Send(messageConfig))
								return
							}

							// Save downloaded file to the FS.
							err = ioutil.WriteFile(uploadPath, fileData, 0600)
							if err != nil {
								// Prepare response message for error.
								err = errors.Wrap(err, "failed to write file content")
								messageConfig := newMessageConfig(update.Message, err.Error())
								logSendMessage(bot.Send(messageConfig))
								return
							}

							// Specified file uploaded to file system.
							messageConfig := newMessageConfig(update.Message, "Uploaded: "+uploadPath)
							logSendMessage(bot.Send(messageConfig))
						}(update.Message, chats[update.Message.Chat.ID].UploadPath)
					}
				}

			// Handle shell command.
			default:
				// Switch chat state back to initial to rule out state traps.
				chats[update.Message.Chat.ID].State = ChatStateInitial

				if checkLogin(chats, update.Message, bot) {
					go func(message *tgbotapi.Message) {
						output, err := executeInShell(update.Message.Text)
						output = strings.Trim(output, "\n")

						writer := MessagesWriter{
							maxMessageLength: 4096,
							maxMessagesCount: 10,
							newMessageConfig: func() tgbotapi.MessageConfig {
								return newMessageConfig(message, "")
							},
						}
						writer.Write("Output:", "bold")
						writer.Write("\n", "")
						writer.Write(output, "code")

						if err != nil {
							// Prepare error response message for command run.
							writer.Write("\n\n", "")
							writer.Write("Error:", "bold")
							writer.Write("\n", "")
							writer.Write(err.Error(), "code")
						}

						// Send prepared messages.
						for _, messageConfig := range writer.Messages() {
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

// getFileBytes returns file bytes by path.
func getFileBytes(path string) (*tgbotapi.FileBytes, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read file")
	}

	return &tgbotapi.FileBytes{
		Name:  filepath.Base(path),
		Bytes: data,
	}, nil
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

	return strings.ToValidUTF8(string(output), ""), nil
}

// MessagesWriter writes messages with splitting and markup.
type MessagesWriter struct {
	maxMessageLength int
	maxMessagesCount int
	newMessageConfig func() tgbotapi.MessageConfig
	messageConfigs   []tgbotapi.MessageConfig
	messageEntities  []tgbotapi.MessageEntity
	stringBuilder    strings.Builder
}

// Write adds specified message text with markup.
func (mw *MessagesWriter) Write(largeMessage string, format string) {
	for largeMessage != "" {
		chunkMessage := largeMessage
		builderRuneCount := mw.getStringRuneCount(mw.stringBuilder.String())
		messageRuneCount := mw.getStringRuneCount(chunkMessage)

		// If message is larger than available size, then pick part of it
		if builderRuneCount+messageRuneCount > mw.maxMessageLength {
			// Split original message to right size chunks.
			freeSpace := mw.maxMessageLength - builderRuneCount
			messageRunes := []rune(chunkMessage)
			chunkMessage = string(messageRunes[:freeSpace])
			largeMessage = string(messageRunes[freeSpace:])
		} else {
			// Everything is written.
			largeMessage = ""
		}

		// Store metadata to entities slice.
		if format != "" {
			mw.messageEntities = append(mw.messageEntities, tgbotapi.MessageEntity{
				Type:   format,
				Offset: mw.getUTF16BytesCount(mw.stringBuilder.String()),
				Length: mw.getUTF16BytesCount(chunkMessage),
			})
		}

		// Store message to string builder.
		mw.stringBuilder.WriteString(chunkMessage)

		// Flush accumulated data when it was an overflow.
		if builderRuneCount+messageRuneCount > mw.maxMessageLength {
			mw.flush()
		}
	}
}

// flush flushes accumulated data to messages.
func (mw *MessagesWriter) flush() {
	// When max messages limit achieved.
	if len(mw.messageConfigs) >= mw.maxMessagesCount {
		mw.stringBuilder = strings.Builder{}
		mw.messageEntities = nil
		return
	}

	messageConfig := mw.newMessageConfig()
	messageConfig.Text = mw.stringBuilder.String()
	messageConfig.Entities = mw.messageEntities

	mw.messageConfigs = append(mw.messageConfigs, messageConfig)
	mw.stringBuilder = strings.Builder{}
	mw.messageEntities = nil
}

// Messages returns accumulated message configs.
func (mw *MessagesWriter) Messages() []tgbotapi.MessageConfig {
	if mw.stringBuilder.Len() != 0 {
		mw.flush()
	}
	return mw.messageConfigs
}

// getUTF16BytesCount returns count of bytes for UTF-16 version of `utf8string`.
func (mw *MessagesWriter) getUTF16BytesCount(utf8string string) int {
	return len(utf16.Encode([]rune(utf8string)))
}

// getStringRuneCount returns count of runes for UTF-8 string in `utf8string`.
func (mw *MessagesWriter) getStringRuneCount(utf8string string) int {
	return utf8.RuneCountInString(utf8string)
}
