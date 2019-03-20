package main

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"regexp"
	"strings"
)

var (
	commandPrefix string
	botID         string
	discord       *discordgo.Session
)

// func main() {
// 	startDiscord()
// 	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
// 	sc := make(chan os.Signal, 1)
// 	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
// 	<-sc
// 	closeDiscord()
// }

func closeDiscord() {
	discord.Close()
}

func startDiscord() {
	var err error
	discord, err = discordgo.New("Bot " + discordBotToken)
	errCheck("error creating discord session", err)
	user, err := discord.User("@me")
	errCheck("error retrieving account", err)

	botID = user.ID
	// 	dgBotSession.AddHandler(callbacks.Ready)            // Connection established with Discord
	discord.AddHandler(commandHandler)
	discord.AddHandler(func(discord *discordgo.Session, ready *discordgo.Ready) {
		err = discord.UpdateStatus(0, "Waitin for Puddin...")
		if err != nil {
			fmt.Println("Error attempting to set my status")
		}
		servers := discord.State.Guilds
		fmt.Printf("PuddinWatcherBot has started on %d servers\n", len(servers))
	})

	err = discord.Open()
	errCheck("Error opening connection to Discord", err)

	commandPrefix = "!"
}

func errCheck(msg string, err error) {
	if err != nil {
		fmt.Printf("%s: %+v", msg, err)
		panic(err)
	}
}

var userNameRegex = regexp.MustCompile("[a-zA-Z0-9_]+")

func commandHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	user := m.Author
	if user.ID == botID || user.Bot {
		// Do nothing because a bot is talking
		return
	}

	// content := m.Content

	fmt.Printf("Message: %+v || From: %s\n", m.Message, m.Author)

	// If the message is "ping" reply with "Pong!"
	if m.Content == "ping" {
		s.ChannelMessageSend(m.ChannelID, "Pong!")
	}

	// If the message is "pong" reply with "Ping!"
	if m.Content == "pong" {
		s.ChannelMessageSend(m.ChannelID, "Ping!")
	}

	if strings.ToLower(m.Content) == commandPrefix+"roomcount" {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("There are currently %d rooms online", onlineRoomCount))
	}

	split := strings.Split(m.Content, " ")

	fmt.Println(split)
	if len(split) < 2 {
		return
	}

	if split[0] == commandPrefix+"status" {
		statusCmd(s, m, split[1:])
	}

	if split[0] == commandPrefix+"viewing" {
		viewingCmd(s, m, split[1:])
	}
}
