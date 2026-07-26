package utils_test

import (
	"testing"

	"hidden-prompt-backend/pkg/utils"

	"github.com/stretchr/testify/require"
)

type scoreOnly struct {
	IntentScore int `json:"intent_score"`
}

func Test_ExtractJSON(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want int
	}{
		{
			name: "plain json",
			raw:  `{"intent_score": 80}`,
			want: 80,
		},
		{
			name: "plain json with surrounding whitespace",
			raw:  "  \n  " + `{"intent_score": 42}` + "  \n  ",
			want: 42,
		},
		{
			name: "triple backtick with json tag",
			raw:  "```json\n{\"intent_score\": 55}\n```",
			want: 55,
		},
		{
			name: "triple backtick without tag",
			raw:  "```\n{\"intent_score\": 33}\n```",
			want: 33,
		},
		{
			name: "single backtick",
			raw:  "`{\"intent_score\": 17}`",
			want: 17,
		},
		{
			name: "prose before and after a fenced block",
			raw:  "Sure, here you go:\n```json\n{\"intent_score\": 91}\n```\nHope that helps!",
			want: 91,
		},
		{
			name: "decoy brace before the real json object",
			raw:  `The format looks like {example}. Here is the real answer: {"intent_score": 64}`,
			want: 64,
		},
		{
			name: "trailing comma",
			raw:  `{"intent_score": 12,}`,
			want: 12,
		},
		{
			name: "trailing comma inside fenced block",
			raw:  "```json\n{\"intent_score\": 8,}\n```",
			want: 8,
		},
		{
			name: "multiple fenced blocks, only the second is valid json",
			raw:  "```\nnot json at all\n```\nWait, actually:\n```\n{\"intent_score\": 77}\n```",
			want: 77,
		},
		{
			name: "nested object containing braces inside a string value",
			raw:  `{"intent_score": 5, "note": "looks like {this}"}`,
			want: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := utils.ExtractJSON[scoreOnly](tt.raw)
			require.NoError(t, err)
			require.Equal(t, tt.want, got.IntentScore)
		})
	}
}

func Test_ExtractJSON_Failures(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{name: "empty response", raw: ""},
		{name: "whitespace only", raw: "   \n\t  "},
		{name: "no json anywhere", raw: "I cannot answer that question."},
		{name: "unbalanced braces", raw: "{\"intent_score\": 10"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := utils.ExtractJSON[scoreOnly](tt.raw)
			require.Error(t, err)
		})
	}
}
