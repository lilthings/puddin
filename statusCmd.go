package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func statusCmd(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if len(args) != 1 {
		fmt.Printf("incorrect num args (%d): %s\n", len(args), strings.Join(args, " "))
		return
	}

	if userNameRegex.MatchString(args[0]) {
		room := strings.ToLower(args[0])
		apiUrl := fmt.Sprintf("https://chaturbate.com/api/chatvideocontext/%s/", room)
		res, err := http.Get(apiUrl)
		if err != nil {
			fmt.Println(err)
			s.ChannelMessageSend(m.ChannelID, "Error! "+err.Error())
			return
		}
		contents, err := ioutil.ReadAll(res.Body)
		if err != nil {
			fmt.Println(err)
			s.ChannelMessageSend(m.ChannelID, "Error! "+err.Error())
			return
		}
		res.Body.Close()

		if res.StatusCode == http.StatusUnauthorized {
			fmt.Println("401!d " + room)
			s.ChannelMessageSend(m.ChannelID, "Password required :(")
			return
		}

		var cvc ChatVideoContext
		err = json.Unmarshal(contents, &cvc)
		if err != nil {
			fmt.Println(err)
			s.ChannelMessageSend(m.ChannelID, "Error! "+err.Error())
			return
		}

		regViewers, anonViewers, err := getViewerCount(cvc.BroadcasterUsername)
		if err != nil {
			fmt.Println(err)
			// s.ChannelMessageSend(m.ChannelID, "Error! "+err.Error())
			// return
		}

		s.ChannelMessageSendEmbed(m.ChannelID, &discordgo.MessageEmbed{
			URL:   "https://chaturbate.com/" + room,
			Title: room,
			Color: 0xff008c,
			// Footer: &discordgo.MessageEmbedFooter{Text: "Made using the discordgo library"},
			Image: &discordgo.MessageEmbedImage{
				URL: fmt.Sprintf("https://roomimg.stream.highwebmedia.com/ri/%s.jpg?%d", room, time.Now().Unix()),
			},
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Status",
					Value:  cvc.RoomStatus,
					Inline: true,
				},
				{
					Name:   "Viewers",
					Value:  fmt.Sprintf("%d", anonViewers+regViewers),
					Inline: true,
				},
				{
					Name:   "Registered Viewers",
					Value:  fmt.Sprintf("%d", regViewers),
					Inline: true,
				},
				{
					Name:   "Anon Viewers",
					Value:  fmt.Sprintf("%d", anonViewers),
					Inline: true,
				},
				{
					Name:   "Title",
					Value:  stripTitleTags(cvc.RoomTitle),
					Inline: false,
				},
			},
		})
		return
	}
}
