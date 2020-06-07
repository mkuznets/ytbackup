package browser

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/go-pkgz/repeater"
)

type Browser struct {
	executable string
	debugPort  int
	execArgs   []string
}

func New(executable, dataDir string, port int, extraArgs map[string]string) (*Browser, error) {
	args := extraArgs
	if args == nil {
		args = make(map[string]string)
	}

	if dataDir != "" {
		fi, err := os.Stat(dataDir)
		if err != nil {
			return nil, err
		}
		if !fi.IsDir() {
			return nil, errors.New("invalid data directory")
		}
		args["--user-data-dir"] = dataDir
	}
	args["--headless"] = ""
	args["--window-size"] = "1920,1080"
	args["--remote-debugging-port"] = fmt.Sprintf("%d", port)

	execArgs := make([]string, 0, 2*len(args))
	for k, v := range args {
		arg := k
		if v != "" {
			arg = fmt.Sprintf("%s=%s", arg, v)
		}
		execArgs = append(execArgs, arg)
	}

	return &Browser{
		executable: executable,
		debugPort:  port,
		execArgs:   execArgs,
	}, nil
}

func (b *Browser) Run(ctx context.Context) error {
	ctxTimeout, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	log.Debug().Str("cmd", b.executable).Strs("args", b.execArgs).Msg("Running browser")

	execCmd := exec.CommandContext(ctxTimeout, b.executable, b.execArgs...)
	if err := execCmd.Run(); err != nil {
		if errors.Is(ctxTimeout.Err(), context.Canceled) {
			return nil
		}
		return err
	}
	return nil
}

func (b *Browser) DebugURL(ctx context.Context) (string, error) {
	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	endpoint := fmt.Sprintf("http://127.0.0.1:%d/json/version", b.debugPort)

	req, err := http.NewRequestWithContext(ctxTimeout, "GET", endpoint, nil)
	if err != nil {
		return "", err
	}

	var resp *http.Response
	err = repeater.NewDefault(10, time.Second).Do(ctxTimeout, func() (err error) {
		resp, err = http.DefaultClient.Do(req)
		return
	})
	if err != nil {
		return "", fmt.Errorf("could not connect to the browser: %v", err)
	}

	var data struct {
		URL string `json:"webSocketDebuggerUrl"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}

	return data.URL, nil
}

func (b *Browser) Do(ctx context.Context, f func(ctx context.Context, url string) error) error {
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	wg.Add(1)

	go func() {
		defer wg.Done()
		if err := b.Run(ctx); err != nil {
			log.Err(err).Msg("Browser error")
		}
	}()

	url, err := b.DebugURL(ctx)
	if err != nil {
		return err
	}

	return f(ctx, url)
}
