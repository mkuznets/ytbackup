package main

import (
	"mkuznets.com/go/ytbackup/internal/ytbackup"
	"mkuznets.com/go/ytbackup/internal/ytbackup/check"
	"mkuznets.com/go/ytbackup/internal/ytbackup/start"
)

type Options struct {
	Common  *ytbackup.Options        `group:"Common Options"`
	Start   *start.Command           `command:"start" description:"Start pulling sources and downloading videos"`
	Setup   *ytbackup.SetupCommand   `command:"setup" description:"Configure OAuth token for Youtube API"`
	Import  *ytbackup.ImportCommand  `command:"import" description:"Import videos from Google's takeout JSON files"`
	List    *ytbackup.ListCommand    `command:"list" description:"List videos"`
	Check   *check.Command           `command:"check" description:"Data integrity checks"`
	Add     *ytbackup.AddCommand     `command:"add"  description:"Add one or more videos by ID"`
	Version *ytbackup.VersionCommand `command:"version" description:"Show version"`
}
