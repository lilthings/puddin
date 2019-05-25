package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"gopkg.in/olivere/elastic.v6"
)

var foundViewer = make(map[string]bool)
var onlineViewer = make(map[string]bool)

const watchlist = "viewerWatchlist.txt"

func init() {
	b, err := ioutil.ReadFile(watchlist)
	if err != nil {
		panic(err)
	}
	username := strings.Split(string(b), "\n")
	for _, value := range username {
		name := strings.TrimSpace(value)
		if name == "" {
			continue
		}
		foundViewer[name] = false
		onlineViewer[name] = false
	}
}

func logViewers(affId string, client *elastic.Client, ctx context.Context) {
	for {
		bulk := client.Bulk()
		t := time.Now()
		regionBlocked := 0

		viewerChan := make(chan roomViewer)
		roomChan := make(chan *OnlineModel)
		wg := sync.WaitGroup{}

		onlineModels, err := getOnlineRooms(affId)
		if err != nil {
			fmt.Println(err)
			goto sleep
		}

		go func() {
			for viewer := range viewerChan {
				if o, ok := onlineViewer[viewer.Username]; ok {
					if !o {
						_, _ = discord.ChannelMessageSend(viewerNotificationChannelId, viewer.Username+" is now online")
					}
					foundViewer[viewer.Username] = true
					onlineViewer[viewer.Username] = true
				}

				viewer.BatchTime = t
				bulk.Add(elastic.NewBulkIndexRequest().
					Index(viewerIndexName).
					Type("_doc").
					Doc(viewer))

				if bulk.EstimatedSizeInBytes() > 80*1e6 {
					_, err := bulk.Do(ctx)
					if err != nil {
						fmt.Println(err)
					}
				}
			}
		}()

		for worker := 0; worker < 10; worker++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for room := range roomChan {
					reg, _, err := getViewers(room.Username)
					if err != nil {
						if err != errRegionBlocked {
							fmt.Println(room.Username, err)
						} else {
							regionBlocked++
						}
					}
					err = downloadRoomThumb("./thumbs", room.Gender, room.Username)
					if err != nil {
						fmt.Printf("error fetching thumb for %s: %s\n", room.Username, err)
					}
					for _, value := range reg {
						viewerChan <- value
					}
				}
			}()
		}

		for _, value := range onlineModels {
			roomChan <- &value
		}

		close(roomChan)
		wg.Wait()
		close(viewerChan)

		for name, online := range onlineViewer {
			if online && !foundViewer[name] {
				onlineViewer[name] = false
				_, _ = discord.ChannelMessageSend(viewerNotificationChannelId, name+" is now offine")
			}
			// clear for next pass
			foundViewer[name] = false
		}
	sleep:
		if bulk.NumberOfActions() > 0 {
			_, err := bulk.Do(ctx)
			if err != nil {
				fmt.Println(err)
			}
		}

		u := time.Until(t.Add(10 * time.Minute))
		fmt.Printf("%d rooms are region blocked\n", regionBlocked)
		fmt.Printf("Sleeping %s until next viewer check\n", u)
		time.Sleep(u)
	}
}

func viewingCmd(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if len(args) != 1 {
		s := fmt.Sprintf("incorrect num args (%d): %s\n", len(args), strings.Join(args, " "))
		_, _ = discord.ChannelMessageSend(m.ChannelID, s)
		return
	}

	if userNameRegex.MatchString(args[0]) {
		viewer := strings.ToLower(args[0])

		usrFilter := elastic.NewTermQuery("username", viewer)
		recentFilter := elastic.NewRangeQuery("batch_time").Gte("now-20m").Lte("now")
		query := elastic.NewBoolQuery().Filter(usrFilter, recentFilter)

		dateHisto := elastic.NewDateHistogramAggregation().
			Interval("10m").
			Order("_key", false).
			Field("batch_time").
			MinDocCount(1)

		roomAgg := elastic.NewTermsAggregation().
			Field("room")

		dateHisto = dateHisto.SubAggregation("room", roomAgg)

		search := esClient.Search(viewerIndexName).
			Query(query).
			Aggregation("batch_time", dateHisto)

		res, err := search.Do(context.Background())
		if err != nil {
			_, _ = discord.ChannelMessageSend(m.ChannelID, "Error getting viewed rooms")
			return
		}

		dhi, ok := res.Aggregations.DateHistogram("batch_time")
		if !ok {
			_, _ = discord.ChannelMessageSend(m.ChannelID, "Error getting viewed rooms")
			return
		}

		var viewing []string
		if len(dhi.Buckets) > 0 {
			bucket := dhi.Buckets[0]
			ti, ok := bucket.Terms("room")
			if !ok {
				_, _ = discord.ChannelMessageSend(m.ChannelID, "Error getting viewed rooms")
				return
			}
			for _, value := range ti.Buckets {
				room, ok := value.Key.(string)
				if !ok {
					_, _ = discord.ChannelMessageSend(m.ChannelID, "Error getting viewed rooms")
					return
				}
				viewing = append(viewing, room)
			}
		}

		s := fmt.Sprintf("%s was recently seen viewing %d rooms\n----\n%s",
			viewer,
			len(viewing),
			"```\n"+strings.Join(viewing, "\n")+"\n```",
		)

		_, _ = discord.ChannelMessageSend(m.ChannelID, s)

		return
	}
}

func alertViewerCmd(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if len(args) != 1 {
		s := fmt.Sprintf("incorrect num args (%d): %s\n", len(args), strings.Join(args, " "))
		_, _ = discord.ChannelMessageSend(m.ChannelID, s)
		return
	}

	if userNameRegex.MatchString(args[0]) {
		name := strings.ToLower(args[0])

		foundViewer[name] = false
		onlineViewer[name] = false
		_, _ = discord.ChannelMessageSend(m.ChannelID, name+" is now being watched")

		var watching []string
		for user := range onlineViewer {
			watching = append(watching, user)
		}
		watching = append(watching, "")

		err := ioutil.WriteFile(watchlist, []byte(strings.Join(watching, "\n")), 0777)
		if err != nil {
			_, _ = discord.ChannelMessageSend(m.ChannelID, "error saving watchlist")
		}

		return
	}
}

func stopViewerCmd(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if len(args) != 1 {
		s := fmt.Sprintf("incorrect num args (%d): %s\n", len(args), strings.Join(args, " "))
		_, _ = discord.ChannelMessageSend(m.ChannelID, s)
		return
	}

	if userNameRegex.MatchString(args[0]) {
		name := strings.ToLower(args[0])

		delete(foundViewer, name)
		delete(onlineViewer, name)
		_, _ = discord.ChannelMessageSend(m.ChannelID, name+" is no longer being watched")

		var watching []string
		for user := range onlineViewer {
			watching = append(watching, user)
		}
		watching = append(watching, "")

		err := ioutil.WriteFile(watchlist, []byte(strings.Join(watching, "\n")), 0777)
		if err != nil {
			_, _ = discord.ChannelMessageSend(m.ChannelID, "error saving watchlist")
		}

		return
	}
}

func trackingCmd(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	var watching []string
	for user := range onlineViewer {
		watching = append(watching, user)
	}
	watching = append(watching, "")

	str := "currently watching\n```\n" + strings.Join(watching, "\n") + "```"
	_, _ = discord.ChannelMessageSend(m.ChannelID, str)
}

func downloadRoomThumb(basePath string, gender string, roomName string) error {
	if len(roomName) < 1 {
		return errors.New("invalid room name")
	}
	t := time.Now().Unix()
	fileUrl := fmt.Sprintf("https://roomimg.stream.highwebmedia.com/ri/%s.jpg?%d", roomName, t)
	fileDir := path.Join(basePath, gender, roomName[0:1], roomName)
	err := os.MkdirAll(fileDir, 0666)
	if err != nil {
		return err
	}
	filePath := path.Join(fileDir, fmt.Sprintf("%s_%d.jpg", roomName, t))
	if err := DownloadFile(filePath, fileUrl); err != nil {
		return err
	}

	return nil
}

// DownloadFile will download a url to a local file. It's efficient because it will
// write as it downloads and not load the whole file into memory.
func DownloadFile(filepath string, url string) error {
	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}
