package main

import "time"

type Session struct {
	Username string `json:"username"`
	ShowType string `json:"show_type"`
	Gender   string `json:"gender"`
	Location string `json:"location"`
	Birthday string `json:"birthday"`

	AverageViewers int64 `json:"average_viewers"`
	MaxViewers     int64 `json:"max_viewers"`

	StartFollowers int64 `json:"start_followers"`
	EndFollowers   int64 `json:"end_followers"`
	MinFollowers   int64 `json:"min_followers"`
	MaxFollowers   int64 `json:"max_followers"`
	DeltaFollowers int64 `json:"delta_followers"`

	StartTime   time.Time     `json:"start_time"`
	EndTime     time.Time     `json:"end_time"`
	DurationNs  time.Duration `json:"duration_ns"`
	DurationStr string        `json:"duration_str"`

	StartRank int64 `json:"start_rank"`
	EndRank   int64 `json:"end_rank"`
	MinRank   int64 `json:"min_rank"`
	MaxRank   int64 `json:"max_rank"`

	viewersAvgTotal int64
	viewersAvgCount int64
}
