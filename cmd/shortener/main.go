package main

import (
	"context"
	_ "net/http/pprof"

	"github.com/anatolyi0311/cupurl/internal/app"
	"github.com/sirupsen/logrus"
)

func main() {
	ctx := context.Background()

	a, err := app.NewApp(ctx)
	if err != nil {
		logrus.Fatalf("failed to init app: %s", err.Error())
	}

	a.Run()
}
