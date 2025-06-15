package internal

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

var startTime time.Time

var BotToken string

func checkNilErr(e error) {
	if e != nil {
		log.Fatal("Error message")
	}
}

func formatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%02dh %02dm %02ds", h, m, s)
}

func Run() {

	discord, err := discordgo.New("Bot " + BotToken)
	checkNilErr(err)

	discord.AddHandler(newMessage)

	discord.Open()
	defer discord.Close()

	startTime = time.Now()

	log.Printf("Logged in as %s#%s", discord.State.User.Username, discord.State.User.Discriminator)
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

}

func newMessage(discord *discordgo.Session, message *discordgo.MessageCreate) {
	if message.Author.ID == discord.State.User.ID {
		return
	}

	switch {
	case strings.Contains(message.Content, "!help"):
		helpMsg := "``!q (!queue) [mode] to queue up in one of supported modes.\n" +
			"!e (!exit, !dequeue, !leave, !dq) to leave the queue or match.\n" +
			"!r (!result) [win/loss/w/l] to report the result of your current game.\n" +
			"!lb (!leaderboard, !top) [mode] to show the leaderboard with best players in chosen mode, based on their ELO.\n" +
			"!h (!history) to view the history of your latest 10 matches.\n" +
			"!elo [mode] to get a graph with your ELO changes over time.\n" +
			"!myelo to get your ELO numbers from all of the modes.\n" +
			"!balance to get your tokens balance. Tokens can be used for bets.\n" +
			"!bet [amount] [member] to bet chosen amount of tokens on one of the players who are currently in a match. You can only bet once on an ongoing match.\n" +
			"!bet_history to get your history of bets.\n" +
			"!m (!matches) to see the list of currently ongoing matches.``"
		discord.ChannelMessageSend(message.ChannelID, helpMsg)
	case strings.Contains(message.Content, "!uptime"):
		uptime := time.Since(startTime)
		msg := fmt.Sprintf("Bot uptime: %s",
			formatDuration(uptime),
		)
		discord.ChannelMessageSend(message.ChannelID, msg)
	}
}
