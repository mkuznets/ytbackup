package browser

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/go-pkgz/repeater"
	"github.com/mitchellh/go-homedir"
)

type Browser struct {
	executable string
	debugPort  int
	execArgs   []string
}

func New(executable string, dataDir string, port int, extraArgs map[string]string) (*Browser, error) {
	args := extraArgs
	if args == nil {
		args = make(map[string]string)
	}

	if dataDir != "" {
		absDataDir, err := homedir.Expand(dataDir)
		if err != nil {
			return nil, err
		}
		fi, err := os.Stat(absDataDir)
		if err != nil {
			return nil, err
		}
		if !fi.IsDir() {
			return nil, errors.New("invalid data directory")
		}
		args["--user-data-dir"] = absDataDir
	}
	args["--headless"] = ""
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

	log.Printf("running: %s %s", b.executable, strings.Join(b.execArgs, " "))

	execCmd := exec.CommandContext(ctxTimeout, b.executable, b.execArgs...)
	if err := execCmd.Run(); err != nil {
		if errors.Is(ctxTimeout.Err(), context.Canceled) {
			return nil
		}
		log.Print("browser error: %v", err)
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
		Url string `json:"webSocketDebuggerUrl"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}

	return data.Url, nil
}

func (b *Browser) Do(ctx context.Context, f func(ctx context.Context, url string) error) error {
	var wg sync.WaitGroup
	defer wg.Wait()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		wg.Add(1)
		defer wg.Done()
		if err := b.Run(ctx); err != nil {
			log.Print(err)
		}
	}()

	url, err := b.DebugURL(ctx)
	if err != nil {
		return err
	}

	return f(ctx, url)
}