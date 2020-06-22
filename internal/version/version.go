package version

import (
	"fmt"
	"runtime"
	"strings"
	"time"
)

var (
	version   = ""
	revision  = ""
	buildTime = ""
)

func Version() string {
	var b strings.Builder

	// noinspection GoBoolExpressions
	if version != "" { // nolint
		b.WriteString(version)
		b.WriteByte('\n')
	}

	// noinspection GoBoolExpressions
	if buildTime != "" { // nolint
		localBuildTime := buildTime
		bt, err := time.Parse("2006-01-02T15:04:05Z", buildTime)
		if err == nil {
			localBuildTime = bt.Local().Format("2006-01-02 15:04:05 MST")
		}
		b.WriteString(fmt.Sprintf("- Built with %s at %s\n", runtime.Version(), localBuildTime))
	}

	// noinspection GoBoolExpressions
	if revision != "" { // nolint
		b.WriteString(fmt.Sprintf("- Revision: %s\n", revision))
	}

	return b.String()
}
