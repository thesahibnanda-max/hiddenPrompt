package main

import (
	"hidden-prompt-backend/pkg/core/deps"
	env "hidden-prompt-backend/pkg/environment"
)

func main() {
	env.SetDotEnv("./")
	deps.NewApp().Run()
}
