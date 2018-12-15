package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/olivere/elastic"
	"io/ioutil"
	"net/http"
	"time"
)

func watchOnlineRooms(affId string, client *elastic.Client, ctx context.Context) {
	var lastInPasswordShow time.Time
	for {
		bulk := client.Bulk()

		t := time.Now()

		url := "https://chaturbate.com/affiliates/api/onlinerooms/?format=json&wm=" + affId
		response, err := http.Get(url)
		if err != nil {
			fmt.Printf("%s\n", err)
			goto sleep
		} else {
			contents, err := ioutil.ReadAll(response.Body)
			if err != nil {
				fmt.Printf("%s\n", err)
				goto sleep
			}
			response.Body.Close()

			var onlineModels OnlineModels
			err = json.Unmarshal(contents, &onlineModels)
			if err != nil {
				fmt.Printf("%s\n", err)
				goto sleep
			}

			fmt.Printf("%d currently online models being indexed\n", len(onlineModels))

			foundPuddin := false
			for _, value := range onlineModels {
				switch value.CurrentShow {
				case "private":
					// NumUsers is stuck at last count when private started
					value.NumUsers = 0
				case "away":
					// NumUsers is stuck at last count when prior private started
					value.NumUsers = 0
				case "hidden":
					// NumUsers is accurate, do nothing
				case "group":
					// NumUsers seems accurate, do nothing
				case "public":
					// NumUsers is accurate, do nothing
				default:
					fmt.Printf("Unknown CurrentShow %s on model %s.\n", value.CurrentShow, value.Username)
				}

				if value.Username == alertRoom {
					if value.CurrentShow == "public" {
						if !puddinPublic {
							v, err := getViewerCount(alertRoom)
							if err != nil || v == 0 {
								lastInPasswordShow = time.Now()
								fmt.Println("err getting viewers, maybe pwd: ", v, err)
							} else {
								fmt.Println("viewers: ", v)
								if time.Since(lastInPasswordShow) > 5*time.Minute {
									discord.ChannelMessageSendEmbed(notificationChannelId, &discordgo.MessageEmbed{
										URL:   "https://chaturbate.com/" + alertRoom,
										Title: alertRoom + " is now online!",
										Color: 0xff008c,
										// Footer: &discordgo.MessageEmbedFooter{Text: "Made using the discordgo library"},
										Image: &discordgo.MessageEmbedImage{
											URL: fmt.Sprintf("https://roomimg.stream.highwebmedia.com/ri/%s.jpg?%d", alertRoom, time.Now().Unix()),
										},
										Fields: []*discordgo.MessageEmbedField{
											{
												Name:   "Status",
												Value:  "\u200b" + value.CurrentShow,
												Inline: true,
											},
											{
												Name:   "Viewers",
												Value:  "\u200b" + fmt.Sprintf("%d", value.NumUsers),
												Inline: true,
											},
											{
												Name:   "Title",
												Value:  "\u200b" + stripTitleTags(value.RoomSubject),
												Inline: false,
											},
										},
									})
									discord.UpdateStatus(0, "Watchin Puddin :)")

									puddinPublic = true
								}
							}
						}
						foundPuddin = true
					}
				}

				item := elastic.NewBulkIndexRequest().
					Index("rooms").
					Type("_doc").
					Doc(elasticOM{
						Model: value,
						Time:  t,
					})
				bulk.Add(item)

				if bulk.EstimatedSizeInBytes() > 80*1e6 {
					_, err := bulk.Do(ctx)
					if err != nil {
						fmt.Printf("%s\n", err)
						goto sleep
					}
				}
			}
			if !foundPuddin {
				if puddinPublic {
					discord.ChannelMessageSend(notificationChannelId, alertRoom+" room is now offline :(")
					discord.UpdateStatus(0, "Waitin for Puddin...")
				}
				puddinPublic = false
			}
		}

		if bulk.NumberOfActions() > 0 {
			_, err := bulk.Do(ctx)
			if err != nil {
				fmt.Printf("%s\n", err)
				goto sleep
			}
		}

	sleep:
		u := time.Until(t.Add(time.Minute))
		fmt.Printf("Sleeping %s until next check\n", u)
		time.Sleep(u)
	}
}

type OnlineModel struct {
	// No longer populated, ignore it
	// BlockFromCountries  json.RawMessage `json:"block_from_countries,omitempty"`
	// BlockFromStates     json.RawMessage `json:"block_from_states,omitempty"`
	// ChatRoomUrl         string          `json:"chat_room_url,omitempty"`
	// ChatRoomUrlRevShare string          `json:"chat_room_url_revshare,omitempty"`
	// IFrameEmbed         string          `json:"iframe_embed,omitempty"`
	// IFrameEmbedRevShare string          `json:"iframe_embed_rev_share,omitempty"`
	// ImageUrl360x270     string          `json:"image_url_360x270,omitempty"`
	AdsZoneIds            json.RawMessage `json:"ads_zone_ids,omitempty"`
	Age                   int64           `json:"age,omitempty"`
	AllowGroupShows       string          `json:"allow_group_shows,omitempty"`
	AllowPrivateShows     string          `json:"allow_private_shows,omitempty"`
	AppsRunning           string          `json:"apps_running,omitempty"`
	Birthday              string          `json:"birthday,omitempty"`
	BroadcasterGender     string          `json:"broadcaster_gender,omitempty"`
	BroadcasterOnNewChat  string          `json:"broadcaster_on_new_chat,omitempty"`
	BroadcasterUsername   string          `json:"broadcaster_username,omitempty"`
	ChatPassword          string          `json:"chat_password,omitempty"`
	ChatSettings          json.RawMessage `json:"chat_settings,omitempty"`
	ChatUsername          string          `json:"chat_username,omitempty"`
	CurrentShow           string          `json:"current_show,omitempty"`
	DisplayName           string          `json:"display_name,omitempty"`
	EdgeAuth              string          `json:"edge_auth,omitempty"`
	EmailValidated        string          `json:"email_validated,omitempty"`
	FlashHost             string          `json:"flash_host,omitempty"`
	Following             string          `json:"following,omitempty"`
	Gender                string          `json:"gender,omitempty"`
	GroupShowPrice        string          `json:"group_show_price,omitempty"`
	HasStudio             string          `json:"has_studio,omitempty"`
	HiddenMessage         string          `json:"hidden_message,omitempty"`
	HideSatisfactionScore string          `json:"hide_satisfaction_score,omitempty"`
	HlsSource             string          `json:"hls_source,omitempty"`
	ImageUrl              string          `json:"image_url,omitempty"`
	IsAgeVerified         bool            `json:"is_age_verified,omitempty"`
	IsMobile              bool            `json:"is_mobile,omitempty"`
	IsModerator           bool            `json:"is_moderator,omitempty"`
	IsNew                 bool            `json:"is_new,omitempty"`
	IsSupporter           bool            `json:"is_supporter,omitempty"`
	IsWidescreen          bool            `json:"is_widescreen,omitempty"`
	Location              string          `json:"location,omitempty"`
	LowSatisfactionScore  string          `json:"low_satisfaction_score,omitempty"`
	NumFollowed           int64           `json:"num_followed,omitempty"`
	NumFollowedOnline     int64           `json:"num_followed_online,omitempty"`
	NumFollowers          int64           `json:"num_followers,omitempty"`
	NumUsers              int64           `json:"num_users,omitempty"`
	PrivateShowPrice      string          `json:"private_show_price,omitempty"`
	RecommenderHmac       string          `json:"recommender_hmac,omitempty"`
	Recorded              string          `json:"recorded,omitempty"`
	RoomPass              string          `json:"room_pass,omitempty"`
	RoomStatus            string          `json:"room_status,omitempty"`
	RoomSubject           string          `json:"room_subject,omitempty"`
	RoomTitle             string          `json:"room_title,omitempty"`
	SatisfactionScore     string          `json:"satisfaction_score,omitempty"`
	SecondsOnline         int64           `json:"seconds_online,omitempty"`
	ServerName            string          `json:"server_name,omitempty"`
	SpokenLanguages       string          `json:"spoken_languages,omitempty"`
	SpyPrivateShowPrice   string          `json:"spy_private_show_price,omitempty"`
	Tags                  []string        `json:"tags,omitempty"`
	TfaEnabled            string          `json:"tfa_enabled,omitempty"`
	TipsInPast24Hours     string          `json:"tips_in_past_24_hours,omitempty"`
	TokenBalance          string          `json:"token_balance,omitempty"`
	Username              string          `json:"username,omitempty"`
	ViewerGender          string          `json:"viewer_gender,omitempty"`
	ViewerUsername        string          `json:"viewer_username,omitempty"`
	WsChatHost            string          `json:"wschat_host,omitempty"`
}
