package start

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/api/youtube/v3"
	"mkuznets.com/go/ytbackup/internal/index"
	"mkuznets.com/go/ytbackup/internal/utils"
	yt "mkuznets.com/go/ytbackup/internal/youtube"
)

func (cmd *Command) Enqueuer(ctx context.Context) error {
	service, err := yt.NewService(cmd.Ctx, cmd.Config.Youtube.OAuth.Token())
	if err != nil {
		return err
	}

	endpoint := service.Videos.List("snippet,contentDetails")

	return utils.RunEveryInterval(ctx, 5*time.Second, func() error {
		videos, err := cmd.Index.Get(index.StatusNew, 50)
		if err != nil {
			log.Err(err).Msg("Index error")
			time.Sleep(systemErrorDowntime)
		}

		if len(videos) == 0 {
			return nil
		}

		ids := make([]string, 0, len(videos))
		for _, k := range videos {
			ids = append(ids, k.ID)
		}

		endpoint.Id(strings.Join(ids, ","))

		r, err := endpoint.Do()
		if err != nil {
			log.Err(err).Msg("Youtube API error")
			time.Sleep(systemErrorDowntime)
			return nil
		}

		results := make(map[string]*youtube.Video)
		for _, result := range r.Items {
			results[result.Id] = result
		}

		for _, video := range videos {
			result, ok := results[video.ID]
			if !ok {
				video.Status = index.StatusFailed
				video.Reason = "unavailable or deleted"
				continue
			}
			cmd.fromAPIResult(video, result)
		}

		if err := cmd.Index.Put(videos...); err != nil {
			log.Err(err).Msg("Index error")
			time.Sleep(systemErrorDowntime)
			return nil
		}

		logProgress(videos)

		return nil
	})
}

func (cmd *Command) fromAPIResult(video *index.Video, result *youtube.Video) {
	publishedAt, err := time.Parse(time.RFC3339, result.Snippet.PublishedAt)
	if err != nil {
		video.Status = index.StatusFailed
		video.Reason = fmt.Sprintf("could not parse upload time: %v", err)
		return
	}

	dur, err := utils.ParseISO8601(result.ContentDetails.Duration)
	if err != nil {
		video.Status = index.StatusFailed
		video.Reason = fmt.Sprintf("could not parse duration: %v", err)
		return
	}

	video.Meta = &index.Meta{
		Title:        result.Snippet.Title,
		Description:  result.Snippet.Description,
		ChannelID:    result.Snippet.ChannelId,
		ChannelTitle: result.Snippet.ChannelTitle,
		Tags:         result.Snippet.Tags,
		PublishedAt:  publishedAt,
	}
	video.Status = index.StatusEnqueued

	if result.Snippet.LiveBroadcastContent == "live" {
		video.Status = index.StatusSkipped
		video.Reason = "live"
	}

	if dur > cmd.Config.Sources.MaxDuration {
		video.Status = index.StatusSkipped
		video.Reason = "too long"
	}
}

func logProgress(videos []*index.Video) {
	statuses := make(map[string]int)

	for _, video := range videos {
		statuses[string(video.Status)]++
	}

	e := log.Info()
	for st, n := range statuses {
		e = e.Int(st, n)
	}

	e.Msg("Enqueuer")
}
