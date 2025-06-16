package main

import (
	"os"

	"github.com/dukedeobald/TWWh3Ladder-Go/internal"
)

func main() {
	tokenFile := "ladder token.txt"
	BotToken, err := os.ReadFile(tokenFile)
	if err != nil {
		panic(err)
	}
	internal.CreateTables()
	internal.BotToken = string(BotToken)
	internal.Run()
}
