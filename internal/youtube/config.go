package youtube

import (
	"golang.org/x/oauth2"
	"google.golang.org/api/youtube/v3"
	"mkuznets.com/go/ytbackup/pkg/obscure"
)

const (
	clientID     = "1tpWYwsqI8HKDXHnE3wiuthJU8AXQJx1Rx3sW0LHd0OFC-fjYmAPU3-oWgbt9tXpm1SE8hn2zV0EifEx0D-FsuLTmDgti3v_D74r_E-gd3Xs5GZLkA0ahQ" // nolint
	clientSecret = "xgLU2qax3MjRda0GDo_rzoAVdED4kRp4XYLENwR7El7JZDVPpUuy0Q"                                                                 // nolint
)

func NewConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     obscure.MustReveal(clientID),
		ClientSecret: obscure.MustReveal(clientSecret),
		RedirectURL:  "http://127.0.0.1:7798",
		Scopes:       []string{youtube.YoutubeReadonlyScope},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/auth",
			TokenURL: "https://oauth2.googleapis.com/token",
		},
	}
}
