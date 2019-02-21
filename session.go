package main

import "time"

type Session struct {
	Username string `json:"username,omitempty"`
	ShowType string `json:"show_type,omitempty"`
	Gender   string `json:"gender,omitempty"`
	Location string `json:"location,omitempty"`
	Birthday string `json:"birthday,omitempty"`

	AverageViewers int64 `json:"average_viewers,omitempty"`
	MaxViewers     int64 `json:"max_viewers,omitempty"`

	StartFollowers int64 `json:"start_followers,omitempty"`
	EndFollowers   int64 `json:"end_followers,omitempty"`
	MinFollowers   int64 `json:"min_followers,omitempty"`
	MaxFollowers   int64 `json:"max_followers,omitempty"`
	DeltaFollowers int64 `json:"delta_followers,omitempty"`

	StartTime   time.Time     `json:"start_time,omitempty"`
	EndTime     time.Time     `json:"end_time,omitempty"`
	DurationNs  time.Duration `json:"duration_ns,omitempty"`
	DurationStr string        `json:"duration_str,omitempty"`

	StartRank int64 `json:"start_rank,omitempty"`
	EndRank   int64 `json:"end_rank,omitempty"`
	MinRank   int64 `json:"min_rank,omitempty"`
	MaxRank   int64 `json:"max_rank,omitempty"`

	viewersAvgTotal int64
	viewersAvgCount int64
}
