package main

import (
	"context"
	"fmt"
	"gopkg.in/olivere/elastic.v6"
	"sync"
	"time"
)

var foundViewer = false
var onlineViewer = false

func logViewers(affId string, client *elastic.Client, ctx context.Context) {
	for {
		bulk := client.Bulk()
		t := time.Now()
		regionBlocked := 0

		viewerChan := make(chan roomViewer)
		roomChan := make(chan string)
		wg := sync.WaitGroup{}

		onlineModels, err := getOnlineRooms(affId)
		if err != nil {
			fmt.Println(err)
			goto sleep
		}

		go func() {
			for viewer := range viewerChan {
				if viewer.Username == viewerName {
					if foundViewer == false {
						_, _ = discord.ChannelMessageSend(viewerNotificationChannelId, viewerName+" is now online")
					}
					foundViewer = true
					onlineViewer = true
				}

				viewer.BatchTime = t
				bulk.Add(elastic.NewBulkIndexRequest().
					Index("viewers").
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

		for worker := 0; worker < 5; worker++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for username := range roomChan {
					reg, _, err := getViewers(username)
					if err != nil {
						if err != errRegionBlocked {
							fmt.Println(username, err)
						} else {
							regionBlocked++
						}
					}
					for _, value := range reg {
						viewerChan <- value
					}
				}
			}()
		}

		for _, value := range onlineModels {
			roomChan <- value.Username
		}

		close(roomChan)
		wg.Wait()
		close(viewerChan)

		if !foundViewer && onlineViewer {
			onlineViewer = false
			_, _ = discord.ChannelMessageSend(viewerNotificationChannelId, viewerName+" is now offine")
		}
		// clear for next pass
		foundViewer = false

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
