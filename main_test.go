package main

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"reflect"
	"strings"
	"testing"
)

// Test_splitLargeMessage tests splitLargeMessage function.
func Test_splitLargeMessage(t *testing.T) {
	type args struct {
		messageConfig tgbotapi.MessageConfig
	}
	tests := []struct {
		name string
		args args
		want []tgbotapi.MessageConfig
	}{{
		name: "SplitSmallMessage",
		args: args{tgbotapi.MessageConfig{
			Text: strings.Repeat("X", 1000),
		}},
		want: []tgbotapi.MessageConfig{{
			Text: strings.Repeat("X", 1000),
		}},
	}, {
		name: "SplitLargeMessage",
		args: args{tgbotapi.MessageConfig{
			Text: strings.Join([]string{
				strings.Repeat("A", 4096),
				strings.Repeat("B", 4096),
				strings.Repeat("C", 1808),
			}, ""),
		}},
		want: []tgbotapi.MessageConfig{
			{Text: strings.Repeat("A", 4096)},
			{Text: strings.Repeat("B", 4096)},
			{Text: strings.Repeat("C", 1808)},
		},
	}, {
		name: "SplitSmallMessageWithEntities",
		args: args{tgbotapi.MessageConfig{
			Text: strings.Repeat("X", 1000),
			Entities: []tgbotapi.MessageEntity{
				{Type: "bold", Offset: 0, Length: 100},
				{Type: "code", Offset: 100, Length: 900},
			},
		}},
		want: []tgbotapi.MessageConfig{{
			Text: strings.Repeat("X", 1000),
			Entities: []tgbotapi.MessageEntity{
				{Type: "bold", Offset: 0, Length: 100},
				{Type: "code", Offset: 100, Length: 900},
			},
		}},
	}, {
		name: "SplitLargeMessageWithEntities",
		args: args{tgbotapi.MessageConfig{
			Text: strings.Join([]string{
				strings.Repeat("A", 4096),
				strings.Repeat("B", 4096),
				strings.Repeat("C", 1808),
			}, ""),
			Entities: []tgbotapi.MessageEntity{
				{Type: "bold", Offset: 0, Length: 100},
				{Type: "code", Offset: 100, Length: 800},
				{Type: "bold", Offset: 900, Length: 100},
				{Type: "code", Offset: 1000, Length: 8000},
				{Type: "bold", Offset: 9000, Length: 900},
				{Type: "code", Offset: 9900, Length: 10},
			},
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
				Text: strings.Repeat("B", 4096),
				Entities: []tgbotapi.MessageEntity{
					{Type: "code", Offset: 0, Length: 4096},
				},
			},
			{
				Text: strings.Repeat("C", 1808),
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
			if got := splitLargeMessage(tt.args.messageConfig); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("splitLargeMessage() = %v, want %v", got, tt.want)
			}
		})
	}
}
