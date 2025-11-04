package handlers

import (
    "context"
    "time"
)

const dbTimeout = 5 * time.Second

func withDBTimeout() (context.Context, context.CancelFunc) {
    return context.WithTimeout(context.Background(), dbTimeout)
}

