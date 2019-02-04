package main

import (
	"context"
	"fmt"
	"github.com/olivere/elastic"
	"sync"
	"time"
)

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
	sleep:
		if bulk.NumberOfActions() > 0 {
			_, err := bulk.Do(ctx)
			if err != nil {
				fmt.Println(err)
			}
		}

		u := time.Until(t.Add(20 * time.Minute))
		fmt.Printf("%d rooms are region blocked\n", regionBlocked)
		fmt.Printf("Sleeping %s until next viewer check\n", u)
		time.Sleep(u)
	}
}
