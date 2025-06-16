package internal

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

var MODE_MAP = map[string]int{
	"land":       1,
	"l":          1,
	"conquest":   2,
	"c":          2,
	"domination": 3,
	"d":          3,
	"luckytest":  4,
	"lt":         4,
}

var REVERSE_MODE_MAP = map[int]string{1: "land",
	2: "conquest",
	3: "domination",
	4: "luckytest",
}

var mdb *MatchDB

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
	db := InitDB()
	mdb = NewMatchDB(db)

	discord, err := discordgo.New("Bot " + BotToken)
	checkNilErr(err)

	discord.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		newMessage(s, m, mdb)
	})

	discord.Open()
	defer discord.Close()

	startTime = time.Now()

	log.Printf("Logged in as %s#%s", discord.State.User.Username, discord.State.User.Discriminator)
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

}

func newMessage(discord *discordgo.Session, message *discordgo.MessageCreate, mdb *MatchDB) {
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
	case strings.Contains(message.Content, "!s"):
		playerID, err := strconv.ParseInt(message.Author.ID, 10, 64)
		if err != nil {
			discord.ChannelMessageSend(message.ChannelID, "Invalid user ID.")
			return
		}

		queueStatus := mdb.GetQueueStatus(playerID)
		if queueStatus != nil {
			modeName, ok := REVERSE_MODE_MAP[*queueStatus]
			if !ok {
				modeName = "Unknown Mode"
			}
			msg := fmt.Sprintf("You are in the queue for %s mode.", modeName)
			discord.ChannelMessageSend(message.ChannelID, msg)
			return
		}

		opponent, gameModeID := mdb.GetMatchDetails(playerID)
		if opponent != nil && gameModeID != nil {
			modeName, ok := REVERSE_MODE_MAP[*gameModeID]
			if !ok {
				modeName = "Unknown Mode"
			}
			threadID := mdb.GetMatchThread(playerID, *opponent, *gameModeID)
			if threadID != "" {
				threadChannel, err := discord.Channel(threadID)
				if err == nil && threadChannel != nil {
					msg := fmt.Sprintf("%s, you are in a match (%s) against <@%d> in %s.",
						message.Author.Username, modeName, *opponent)
					discord.ChannelMessageSend(message.ChannelID, msg)
					return
				}
			}
			msg := fmt.Sprintf("%s, you are in a match (%s) against <@%d>.", message.Author.Username, modeName, *opponent)
			discord.ChannelMessageSend(message.ChannelID, msg)
			return
		}

		discord.ChannelMessageSend(message.ChannelID, fmt.Sprintf("%s, you are not in queue or in a match.", message.Author.Username))

	}
}
