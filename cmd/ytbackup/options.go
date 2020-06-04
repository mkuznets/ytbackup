package main

import "mkuznets.com/go/ytbackup/internal/ytbackup"

type Options struct {
	Common *ytbackup.Options       `group:"Common Options"`
	Start  *ytbackup.StartCommand  `command:"start" description:""`
	Setup  *ytbackup.SetupCommand  `command:"setup" description:""`
	Import *ytbackup.ImportCommand `command:"import" description:"retrieve videos from Google's takeout JSON files"`
}
