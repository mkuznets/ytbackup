package main

import "mkuznets.com/go/ytbackup/internal/ytbackup"

type Options struct {
	Common *ytbackup.Options      `group:"Common Options"`
	Start  *ytbackup.StartCommand `command:"start" description:""`
	Setup  *ytbackup.SetupCommand `command:"setup" description:""`
}
