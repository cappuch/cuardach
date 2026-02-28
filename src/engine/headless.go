package engine

import (
	"fmt"
	"sync"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

var (
	browserMu   sync.Mutex
	browserInst *rod.Browser
	browserErr  error
	browserOnce sync.Once
)

func getBrowser() (*rod.Browser, error) {
	browserOnce.Do(func() {
		path, found := launcher.LookPath()
		l := launcher.New().
			Headless(true).
			Set("disable-gpu").
			Set("no-sandbox").
			Set("disable-dev-shm-usage").
			Set("disable-background-networking").
			Set("disable-default-apps").
			Set("disable-extensions").
			Set("disable-sync").
			Set("disable-translate").
			Set("metrics-recording-only").
			Set("no-first-run")

		if found {
			l = l.Bin(path)
		}
		u, err := l.Launch()
		if err != nil {
			browserErr = fmt.Errorf("launching headless browser: %w", err)
			return
		}

		browserInst = rod.New().ControlURL(u)
		if err := browserInst.Connect(); err != nil {
			browserErr = fmt.Errorf("connecting to browser: %w", err)
			return
		}
	})
	return browserInst, browserErr
}

func CloseBrowser() {
	browserMu.Lock()
	defer browserMu.Unlock()
	if browserInst != nil {
		browserInst.Close()
		browserInst = nil
	}
}
