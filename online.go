package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"gopkg.in/olivere/elastic.v6"
	"io/ioutil"
	"net/http"
	"time"
)

var onlineRoomCount int

var lastSessionSet = make(map[string]*Session, 6000)
var currSessionSet = make(map[string]*Session, 6000)

func updateSession(room *OnlineModel, rank int64, t time.Time) {
	s, ok := lastSessionSet[room.Username+`\/`+room.CurrentShow]
	if !ok {
		s = &Session{
			Username:        room.Username,
			ShowType:        room.CurrentShow,
			Gender:          room.Gender,
			Location:        room.Location,
			Birthday:        room.Birthday,
			MaxViewers:      room.NumUsers,
			StartFollowers:  room.NumFollowers,
			EndFollowers:    room.NumFollowers,
			MinFollowers:    room.NumFollowers,
			MaxFollowers:    room.NumFollowers,
			StartTime:       t,
			EndTime:         t,
			StartRank:       rank,
			EndRank:         rank,
			MinRank:         rank,
			MaxRank:         rank,
			viewersAvgTotal: room.NumUsers,
			viewersAvgCount: 1,
		}
	} else {
		s.EndFollowers = room.NumFollowers
		s.EndTime = t
		s.EndRank = rank

		setMin(&s.MinFollowers, room.NumFollowers)
		setMin(&s.MinRank, rank)

		setMax(&s.MaxFollowers, room.NumFollowers)
		setMax(&s.MaxViewers, room.NumUsers)
		setMax(&s.MaxRank, rank)

		s.viewersAvgTotal += room.NumUsers
		s.viewersAvgCount++
	}
	currSessionSet[room.Username] = s
}

func finalizeSessions(bulk *elastic.BulkService) {
	oldSessionSet := lastSessionSet
	lastSessionSet = currSessionSet
	currSessionSet = make(map[string]*Session, 6000)

	for k, oldS := range oldSessionSet {
		_, ok := lastSessionSet[k]
		if !ok {
			dur := oldS.EndTime.Sub(oldS.StartTime)
			fmt.Printf("Session %s ended after %s\n", k, dur)
			oldS.Duration = dur
			oldS.AverageViewers = oldS.viewersAvgTotal / oldS.viewersAvgTotal

			item := elastic.NewBulkIndexRequest().
				Index("room_session").
				Type("_doc").
				Doc(oldS)
			bulk.Add(item)

			if bulk.EstimatedSizeInBytes() > 80*1e6 {
				_, err := bulk.Do(context.TODO())
				if err != nil {
					fmt.Printf("%s\n", err)
				}
			}
		}
	}
}

func setMax(target *int64, value int64) {
	if value > *target {
		*target = value
	}
}

func setMin(target *int64, value int64) {
	if value < *target {
		*target = value
	}
}

func watchOnlineRooms(affId string, client *elastic.Client, ctx context.Context) {
	for {
		bulk := client.Bulk()
		foundPuddin := false
		t := time.Now()

		var fRank int64 = 0
		var mRank int64 = 0
		var cRank int64 = 0
		var sRank int64 = 0

		onlineModels, err := getOnlineRooms(affId)
		if err != nil {
			fmt.Println(err)
			goto sleep
		}

		onlineRoomCount = len(onlineModels)
		fmt.Printf("%d currently online rooms being indexed\n", onlineRoomCount)

		for i := 0; i < onlineRoomCount; i++ {
			value := &onlineModels[i]
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

			rank := int64(i + 1)
			var gRank int64
			switch value.Gender {
			case "f":
				fRank++
				gRank = fRank
			case "m":
				mRank++
				gRank = mRank
			case "c":
				cRank++
				gRank = cRank
			case "s":
				sRank++
				gRank = sRank
			default:
				gRank = -1
			}

			updateSession(value, rank, t)

			if value.Username == alertRoom {
				if value.CurrentShow == "public" {
					if !puddinPublic {
						_, _ = discord.ChannelMessageSendEmbed(notificationChannelId, &discordgo.MessageEmbed{
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
						_ = discord.UpdateStatus(0, "Watchin Puddin :)")

						puddinPublic = true
					}
					foundPuddin = true
				}
			}

			item := elastic.NewBulkIndexRequest().
				Index("rooms").
				Type("_doc").
				Doc(elasticOM{
					Model:      *value,
					Time:       t,
					Rank:       rank,
					GenderRank: gRank,
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
				_, _ = discord.ChannelMessageSend(notificationChannelId, alertRoom+" room is now offline :(")
				_ = discord.UpdateStatus(0, "Waitin for Puddin...")
			}
			puddinPublic = false
		}

		finalizeSessions(bulk)

	sleep:
		if bulk.NumberOfActions() > 0 {
			_, err := bulk.Do(ctx)
			if err != nil {
				fmt.Printf("%s\n", err)
			}
		}
		u := time.Until(t.Add(time.Minute))
		fmt.Printf("Sleeping %s until next online room check\n", u)
		time.Sleep(u)
	}
}

func getOnlineRooms(affId string) (OnlineModels, error) {
	url := "https://chaturbate.com/affiliates/api/onlinerooms/?format=json&wm=" + affId
	response, err := http.Get(url)
	if err != nil {
		return nil, err
	} else {
		contents, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return nil, err
		}
		_ = response.Body.Close()

		var onlineModels OnlineModels
		err = json.Unmarshal(contents, &onlineModels)
		if err != nil {
			return nil, err
		}
		return onlineModels, nil
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
	// ImageUrl            string          `json:"image_url,omitempty"`
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
