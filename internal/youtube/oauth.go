package youtube

import (
	"context"
	"errors"
	"log"
	"net/http"
	"net/url"
	"sync"

	"github.com/pkg/browser"
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

func NewToken(ctx context.Context, config *oauth2.Config) (*oauth2.Token, error) {
	redirectURL, err := url.Parse(config.RedirectURL)
	if err != nil {
		return nil, err
	}

	codeCh := make(chan string)
	srv := &http.Server{Addr: ":" + redirectURL.Port()}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		codeCh <- r.FormValue("code")
		w.Header().Set("Content-Type", "text/plain")
		// noinspection GoUnhandledErrorResult
		w.Write([]byte("You can now safely close this browser window.")) // nolint
	})

	wg := &sync.WaitGroup{}
	defer wg.Wait()

	wg.Add(1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("[ERR] ListenAndServe(): %v", err)
		}
		wg.Done()
	}()

	defer func() {
		go stopServer(ctx, srv)
	}()

	// TODO: support manual code entry for remote setups
	// https://developers.google.com/identity/protocols/oauth2/native-app#manual-copypaste
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	go openURL(authURL)

	code := <-codeCh
	return config.Exchange(ctx, code)
}

func openURL(u string) {
	if err := browser.OpenURL(u); err != nil {
		log.Printf("[ERR] browser.OpenURL: %v", err)
	}
}

func stopServer(ctx context.Context, srv *http.Server) {
	if e := srv.Shutdown(ctx); e != nil {
		log.Printf("[ERR] shutdown: %v", e)
	}
}
