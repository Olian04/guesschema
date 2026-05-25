package guesschema_test

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/Olian04/guesschema/pkg/guesschema"
)

func ExampleNew() {
	ctx := context.Background()
	g, err := guesschema.New(
		guesschema.WithReadWindow(time.Second),
		guesschema.WithVariantThreshold(0.1),
	)
	if err != nil {
		panic(err)
	}
	var out bytes.Buffer
	if err := g.Run(ctx, strings.NewReader("{\"a\":1}\n"), &out); err != nil {
		panic(err)
	}
}

func ExampleWithReadWindow() {
	ctx := context.Background()
	g, err := guesschema.New(guesschema.WithReadWindow(5 * time.Second))
	if err != nil {
		panic(err)
	}
	var out bytes.Buffer
	_ = g.Run(ctx, strings.NewReader("{\"a\":1}\n"), &out)
}

func ExampleWithVariantThreshold() {
	ctx := context.Background()
	g, err := guesschema.New(guesschema.WithVariantThreshold(0.5))
	if err != nil {
		panic(err)
	}
	var out bytes.Buffer
	_ = g.Run(ctx, strings.NewReader("{\"a\":1}\n"), &out)
}

func ExampleWithOmitVendorExtensions() {
	ctx := context.Background()
	g, err := guesschema.New(guesschema.WithOmitVendorExtensions())
	if err != nil {
		panic(err)
	}
	var out bytes.Buffer
	_ = g.Run(ctx, strings.NewReader("{\"a\":1}\n"), &out)
}

func ExampleWithStartWindowOnNextMessage() {
	ctx := context.Background()
	g, err := guesschema.New(guesschema.WithStartWindowOnNextMessage())
	if err != nil {
		panic(err)
	}
	var out bytes.Buffer
	_ = g.Run(ctx, strings.NewReader("{\"a\":1}\n"), &out)
}

func ExampleWithLogger() {
	ctx := context.Background()
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	g, err := guesschema.New(guesschema.WithLogger(log))
	if err != nil {
		panic(err)
	}
	var out bytes.Buffer
	_ = g.Run(ctx, strings.NewReader("{\"a\":1}\n"), &out)
}
