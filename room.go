package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

func getViewerCount(room string) (reg int64, anon int64, err error) {
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
		return 0, 0, err
	}

	if res.StatusCode == 404 {
		fmt.Println("404! " + room)
		return 0, 0, os.ErrNotExist
	}

	if res.StatusCode == 401 {
		fmt.Println("401! " + room)
		return 0, 0, os.ErrPermission
	}

	contents, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return 0, 0, err
	}
	res.Body.Close()

	fmt.Println(string(contents))

	split := strings.Split(string(contents), ",")
	if len(split) > 1 {

	}

	i, err := strconv.ParseInt(split[0], 10, 64)
	if err != nil {
		fmt.Println(err)
		return 0, 0, err
	}

	return int64(len(split)), i, nil
}

type ChatVideoContext struct {
	RoomStatus          string `json:"room_status"`
	RoomTitle           string `json:"room_title"`
	BroadcasterUsername string `json:"broadcaster_username"`
	ChatPassword        string `json:"chat_password"`
	RoomPass            string `json:"room_pass"`
	WsChatHost          string `json:"wschat_host"`
}
