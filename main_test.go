package main

import (
	"reflect"
	"strings"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Test_MessageWriter tests MessageWriter type.
func Test_MessageWriter(t *testing.T) {
	type call struct {
		message string
		format  string
	}
	type args struct {
		calls []call
	}
	tests := []struct {
		name string
		args args
		want []tgbotapi.MessageConfig
	}{{
		name: "WriteSmallMessage",
		args: args{[]call{
			{message: strings.Repeat("X", 1000)},
		}},
		want: []tgbotapi.MessageConfig{{
			Text: strings.Repeat("X", 1000),
		}},
	}, {
		name: "WriteLargeMessage",
		args: args{[]call{
			{message: strings.Repeat("A", 4096)},
			{message: strings.Repeat("B", 4096)},
			{message: strings.Repeat("C", 1808)},
		}},
		want: []tgbotapi.MessageConfig{
			{Text: strings.Repeat("A", 4096)},
			{Text: strings.Repeat("B", 4096)},
			{Text: strings.Repeat("C", 1808)},
		},
	}, {
		name: "WriteSmallMessageWithEntities",
		args: args{[]call{
			{message: strings.Repeat("X", 100), format: "bold"},
			{message: strings.Repeat("X", 900), format: "code"},
		}},
		want: []tgbotapi.MessageConfig{{
			Text: strings.Repeat("X", 1000),
			Entities: []tgbotapi.MessageEntity{
				{Type: "bold", Offset: 0, Length: 100},
				{Type: "code", Offset: 100, Length: 900},
			},
		}},
	}, {
		name: "WriteLargeMessageWithEntities",
		args: args{[]call{
			{message: strings.Repeat("A", 100), format: "bold"},
			{message: strings.Repeat("A", 800), format: "code"},
			{message: strings.Repeat("A", 100), format: "bold"},
			{message: strings.Repeat("A", 4000), format: "code"},
			{message: strings.Repeat("B", 4000), format: "code"},
			{message: strings.Repeat("C", 900), format: "bold"},
			{message: strings.Repeat("C", 10), format: "code"},
		}},
		want: []tgbotapi.MessageConfig{
			{
				Text: strings.Repeat("A", 4096),
				Entities: []tgbotapi.MessageEntity{
					{Type: "bold", Offset: 0, Length: 100},
					{Type: "code", Offset: 100, Length: 800},
					{Type: "bold", Offset: 900, Length: 100},
					{Type: "code", Offset: 1000, Length: 3096},
				},
			},
			{
				Text: strings.Join([]string{
					strings.Repeat("A", 904),
					strings.Repeat("B", 3192),
				}, ""),
				Entities: []tgbotapi.MessageEntity{
					{Type: "code", Offset: 0, Length: 904},
					{Type: "code", Offset: 904, Length: 3192},
				},
			},
			{
				Text: strings.Join([]string{
					strings.Repeat("B", 808),
					strings.Repeat("C", 910),
				}, ""),
				Entities: []tgbotapi.MessageEntity{
					{Type: "code", Offset: 0, Length: 808},
					{Type: "bold", Offset: 808, Length: 900},
					{Type: "code", Offset: 1708, Length: 10},
				},
			},
		},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := MessagesWriter{
				maxMessageLength: 4096,
				maxMessagesCount: 10,
			}
			for _, call := range tt.args.calls {
				writer.Write(call.message, call.format)
			}
			if got := writer.Messages(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MessagesWriter.Messages() = %v, want %v", got, tt.want)
			}
		})
	}
}
