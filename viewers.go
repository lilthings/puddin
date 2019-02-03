package main

import (
	"context"
	"fmt"
	"github.com/olivere/elastic"
	"time"
)

func logViewers(affId string, client *elastic.Client, ctx context.Context) {
	for {
		bulk := client.Bulk()
		t := time.Now()
		regionBlocked := 0

		onlineModels, err := getOnlineRooms(affId)
		if err != nil {
			fmt.Println(err)
			goto sleep
		}

		for _, value := range onlineModels {
			reg, _, err := getViewers(value.Username)
			if err != nil {
				if err != errRegionBlocked {
					fmt.Println(value.Username, err)
				} else {
					regionBlocked++
				}
				continue
			}
			for _, value := range reg {
				bulk.Add(elastic.NewBulkIndexRequest().
					Index("viewers").
					Type("_doc").
					Doc(value))

				if bulk.EstimatedSizeInBytes() > 80*1e6 {
					_, err := bulk.Do(ctx)
					if err != nil {
						fmt.Println(err)
						goto sleep
					}
				}
			}
		}

	sleep:
		if bulk.NumberOfActions() > 0 {
			_, err := bulk.Do(ctx)
			if err != nil {
				fmt.Println(err)
			}
		}

		u := time.Until(t.Add(20 * time.Minute))
		fmt.Printf("%d rooms are region blocked\n", regionBlocked)
		fmt.Printf("Sleeping %s until next check\n", u)
		time.Sleep(u)
	}
}
