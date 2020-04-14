package main

import (
	"golang.org/x/time/rate"
)

func NewLimiter(bps int) *rate.Limiter {
	var limit rate.Limit
	if bps != 0 {
		limit = rate.Limit(bps)
	} else {
		limit = rate.Inf
	}
	// I have no idea why, but unless these are multiplied by two, the speeds
	// are half of what they should be
	return rate.NewLimiter(limit*2, bps*2)
}
