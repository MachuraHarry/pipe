package ai

import (
	"fmt"
	"os"
	"sync"
	"time"
)

type TryAIFixLog struct {
	Time     int64  `json:"time"`
	Code     string `json:"code"`
	Original string `json:"original"`
	Fixed    string `json:"fixed"`
	Attempt  int    `json:"attempt"`
	Success  bool   `json:"success"`
}

var (
	tryAILog    = make([]TryAIFixLog, 0, 200)
	tryAILogMu  sync.Mutex
	tryAILogMax = 200
)

func LogTryAIFix(code, original, fixed string, attempt int, success bool) {
	entry := TryAIFixLog{
		Time:     time.Now().Unix(),
		Code:     code,
		Original: original,
		Fixed:    fixed,
		Attempt:  attempt,
		Success:  success,
	}

	tryAILogMu.Lock()
	tryAILog = append(tryAILog, entry)
	if len(tryAILog) > tryAILogMax {
		tryAILog = tryAILog[len(tryAILog)-tryAILogMax:]
	}
	tryAILogMu.Unlock()

	ts := time.Unix(entry.Time, 0).Format("15:04:05")
	mark := "✗ FAIL"
	if success {
		mark = "✓ FIXED"
	}
	fmt.Fprintf(os.Stderr, "[try_ai] %s | %s | attempt %d | \"%s\" → \"%s\" | %s\n",
		ts, code, attempt, original, fixed, mark)
}

func GetTryAILog() []TryAIFixLog {
	tryAILogMu.Lock()
	defer tryAILogMu.Unlock()
	out := make([]TryAIFixLog, len(tryAILog))
	copy(out, tryAILog)
	return out
}
