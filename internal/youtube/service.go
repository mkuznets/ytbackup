package youtube

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

func NewService(ctx context.Context, token *oauth2.Token) (*youtube.Service, error) {
	tokenSource := NewConfig().TokenSource(ctx, token)

	ytopts := []option.ClientOption{
		option.WithTokenSource(tokenSource),
	}

	service, err := youtube.NewService(ctx, ytopts...)
	if err != nil {
		return nil, fmt.Errorf("could not create youtube client: %v", err)
	}

	return service, nil
}

func IsQuotaError(err error) bool {
	if gErr, ok := err.(*googleapi.Error); ok {
		for _, e := range gErr.Errors {
			if e.Reason == "quotaExceeded" {
				return true
			}
		}
	}
	return false
}
