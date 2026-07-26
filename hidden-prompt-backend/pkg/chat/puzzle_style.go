package chatAndAIUsage

import (
	"hidden-prompt-backend/pkg/utils"
)

// puzzleStyle is a top-level content domain assigned to a puzzle-generation
// call by choosePuzzleStyle (chat.go), rather than left to the model's own
// judgment. A prior version of the system prompt asked the model to "favor
// a mix" of question types itself - in practice it reliably collapsed onto
// one heavily-reinforced pattern regardless of that instruction, so
// variety is now enforced in code instead.
type puzzleStyle string

const (
	puzzleStyleLanguageVocabulary puzzleStyle = "Language & Vocabulary"
	puzzleStyleWordplayRiddles    puzzleStyle = "Wordplay & Riddles"
	puzzleStyleGeneralTrivia      puzzleStyle = "General Trivia"
	puzzleStyleMathematics        puzzleStyle = "Mathematics"
	puzzleStylePatternsLogic      puzzleStyle = "Patterns & Logic"
	puzzleStyleEverydayObjects    puzzleStyle = "Everyday Objects & Home"
	puzzleStyleScienceNature      puzzleStyle = "Science & Nature"
	puzzleStyleFood               puzzleStyle = "Food"
	puzzleStyleGeography          puzzleStyle = "Geography"
	puzzleStyleHistory            puzzleStyle = "History"
	puzzleStyleEntertainment      puzzleStyle = "Entertainment & Culture"
	puzzleStyleSports             puzzleStyle = "Sports"
	puzzleStyleTimeCalendar       puzzleStyle = "Time & Calendar"
	puzzleStyleColorsShapes       puzzleStyle = "Colors & Shapes"
	puzzleStyleSuperlatives       puzzleStyle = "Superlatives & Comparisons"
	puzzleStyleEmojiSymbols       puzzleStyle = "Emoji & Symbols"
	puzzleStyleCommonKnowledge    puzzleStyle = "Common Knowledge & Occupations"
	puzzleStyleEverydayReasoning  puzzleStyle = "Everyday Reasoning"
)

// puzzleStyleAngles gives each domain's specific angles, injected into the
// per-call user prompt as a compact hint (see generate_puzzle_user.qtpl).
// The full definitions and worked examples for each domain live only in
// generate_puzzle_system.md - deliberately not duplicated here, so the two
// can never drift out of sync with each other.
var puzzleStyleAngles = map[puzzleStyle]string{
	puzzleStyleLanguageVocabulary: "synonym, antonym, definition, fill-in-the-blank or phrase completion, rhyming, letters (starts/ends/contains/position/length/alphabetical order), unscrambling, spelling, plural/singular, abbreviation, roman numerals, homophones, syllable count",
	puzzleStyleWordplayRiddles:    "double-meaning riddle, simple analogy",
	puzzleStyleGeneralTrivia:      "plain single-hook factual question, no riddle trick",
	puzzleStyleMathematics:        "arithmetic, percentage, fractions/decimals, greater/less comparison, odd/even, prime/composite, square roots, cubes/powers, unit conversion or estimation, counting",
	puzzleStylePatternsLogic:      "number or alphabet sequence, simple logic, true/false fact, odd-one-out, category membership, sorting by a property",
	puzzleStyleEverydayObjects:    "object by function, tool for a task, room by activity, material identification, household item",
	puzzleStyleScienceNature:      "basic science fact, animal classification/sound/baby name, plant part, human body, cause and effect, weather or season association",
	puzzleStyleFood:               "ingredient, dish origin, fruit vs vegetable, food origin, spice, taste",
	puzzleStyleGeography:          "capital city, country, continent, currency, landmark, largest/longest/highest geography fact",
	puzzleStyleHistory:            "well-known historical person, event, invention, or civilization only - nothing obscure",
	puzzleStyleEntertainment:      "well-known movie, character, quote, song, cartoon, video game, or book only - nothing obscure or dated",
	puzzleStyleSports:             "equipment, rules, or a well-known team/player/record/Olympics fact",
	puzzleStyleTimeCalendar:       "calendar fact, day/month/season, simple time math, clock reading",
	puzzleStyleColorsShapes:       "color identification or mixing, shape identification or properties",
	puzzleStyleSuperlatives:       "fastest, biggest, tallest, oldest, heaviest, or closest-type question",
	puzzleStyleEmojiSymbols:       "emoji-clued object, phrase, or movie",
	puzzleStyleCommonKnowledge:    "who does this job, brand to product, holiday, transport, communication",
	puzzleStyleEverydayReasoning:  "common-sense or situational question with exactly one obviously-correct answer - never a judgment call",
}

// newPuzzleStyleMap builds the weighted picker over every domain above,
// equal weight so none can dominate the way one riddle style dominated in
// practice under the old free-form prompt.
func newPuzzleStyleMap() (utils.ThreadSafeWeightedMap[puzzleStyle], error) {
	weights := make(map[puzzleStyle]uint, len(puzzleStyleAngles))
	for style := range puzzleStyleAngles {
		weights[style] = 1
	}
	return utils.NewThreadSafeWeightedMap(weights)
}
