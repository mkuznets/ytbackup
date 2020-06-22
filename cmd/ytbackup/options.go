package main

import (
	"mkuznets.com/go/ytbackup/internal/ytbackup"
	"mkuznets.com/go/ytbackup/internal/ytbackup/check"
	"mkuznets.com/go/ytbackup/internal/ytbackup/start"
)

type Options struct {
	Common  *ytbackup.Options        `group:"Common Options"`
	Start   *start.Command           `command:"start" description:""`
	Setup   *ytbackup.SetupCommand   `command:"setup" description:""`
	Import  *ytbackup.ImportCommand  `command:"import" description:"retrieve videos from Google's takeout JSON files"`
	List    *ytbackup.ListCommand    `command:"list" description:""`
	Check   *check.Command           `command:"check"`
	Add     *ytbackup.AddCommand     `command:"add" description:"Add videos by ID"`
	Version *ytbackup.VersionCommand `command:"version" description:"Show version"`
}
