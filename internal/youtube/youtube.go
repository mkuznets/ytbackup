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
)

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
