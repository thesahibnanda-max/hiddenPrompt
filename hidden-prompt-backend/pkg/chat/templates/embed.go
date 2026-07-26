package templates

import _ "embed"

//go:embed generate_puzzle_system.md
var generatePuzzleSystemPrompt string

//go:embed hint_system.md
var hintSystemPrompt string

//go:embed score_intent_system.md
var scoreIntentSystemPrompt string

func GeneratePuzzleSystemPrompt() string {
	return generatePuzzleSystemPrompt
}

func HintSystemPrompt() string {
	return hintSystemPrompt
}

func ScoreIntentSystemPrompt() string {
	return scoreIntentSystemPrompt
}
