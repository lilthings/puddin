package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

var errRegionBlocked = errors.New("access denied: this room is not available to your region or gender")

func getViewerCount(room string) (reg int64, anon int64, err error) {
	r, a, e := getViewers(room)
	return int64(len(r)), a, e
}

func getViewers(room string) (reg []roomViewer, anon int64, err error) {
	csrf := RandString(32)

	form := url.Values{}
	form.Add("csrfmiddlewaretoken", csrf)
	form.Add("sort_by", "a")
	form.Add("roomname", room)
	form.Add("private", "false")

	req, _ := http.NewRequest("POST", "https://chaturbate.com/api/getchatuserlist/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")
	req.Header.Add("x-csrftoken", csrf)
	req.Header.Add("cookie", "csrftoken="+csrf+";")
	req.Header.Add("referer", "https://chaturbate.com/")
	req.Header.Add("origin", "https://chaturbate.com")
	req.Header.Add("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) "+
		"AppleWebKit/537.36 (KHTML, like Gecko) Chrome/70.0.3538.110 Safari/537.36")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, err
	}

	if res.StatusCode == 404 {
		fmt.Println("404! " + room)
		return nil, 0, os.ErrNotExist
	}

	if res.StatusCode == 401 {
		fmt.Println("401!a " + room)
		return nil, 0, os.ErrPermission
	}
	if res.StatusCode == 400 {
		fmt.Println("400! " + room)
		return nil, 0, os.ErrInvalid
	}

	contents, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, 0, err
	}
	_ = res.Body.Close()

	if len(contents) == 0 {
		return nil, 0, errRegionBlocked
	}
	// fmt.Println(string(contents))

	split := strings.Split(string(contents), ",")
	i, err := strconv.ParseInt(split[0], 10, 64)
	if err != nil {
		fmt.Println(err)
		return nil, 0, err
	}

	var viewers []roomViewer
	t := time.Now()
	if len(split) > 1 {
		viewers = make([]roomViewer, len(split)-1)
		for _, vString := range split[1:] {
			nameSplit := strings.SplitN(vString, "|", 2)
			if len(nameSplit) != 2 {
				continue
			}
			viewers = append(viewers, roomViewer{
				Time:             t,
				Room:             room,
				Username:         nameSplit[0],
				Color:            nameSplit[1],
				RoomAnonViewers:  i,
				RoomRegViewers:   int64(len(split) - 1),
				RoomTotalViewers: int64(len(split)-1) + i,
			})
		}
	}

	return viewers, i, nil
}

type ChatVideoContext struct {
	RoomStatus          string `json:"room_status"`
	RoomTitle           string `json:"room_title"`
	BroadcasterUsername string `json:"broadcaster_username"`
	ChatPassword        string `json:"chat_password"`
	RoomPass            string `json:"room_pass"`
	WsChatHost          string `json:"wschat_host"`
}

type PrivateCVC struct {
	PrivateShowPrice    int64 `json:"private_show_price"`
	AllowPrivateShows   bool  `json:"allow_private_shows"`
	SpyPrivateShowPrice int64 `json:"spy_private_show_price"`
	GroupShowPrice      int64 `json:"group_show_price"`
	AllowGroupShows     bool  `json:"allow_group_shows"`
}

func getPrivateCVC(room string) (*PrivateCVC, error) {
	room = strings.ToLower(room)
	apiUrl := fmt.Sprintf("https://chaturbate.com/api/chatvideocontext/%s/", room)
	res, err := http.Get(apiUrl)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	contents, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	res.Body.Close()

	if res.StatusCode == http.StatusUnauthorized {
		// fmt.Println("401!b " + room)
		return nil, errRegionBlocked
	}

	cvc := &PrivateCVC{}
	err = json.Unmarshal(contents, cvc)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	return cvc, nil
}
