package main

import "testing"

func TestCampaignRequiresExplicitSend(t *testing.T) {
	env := func(string) string { return "" }
	for _, args := range [][]string{
		{"--kind","broadcast","--text","hello"},
		{"--kind","other","--text","hello","--send"},
		{"--kind","broadcast","--to","U123","--text","hello","--send"},
		{"--kind","multicast","--text","hello","--send"},
	} {
		if err := runCampaign(args, env); err == nil { t.Fatalf("unapproved send: %v",args) }
	}
}
