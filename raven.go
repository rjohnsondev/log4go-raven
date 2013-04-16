// Copyright (C) 2010, Kyle Lemons <kyle@kylelemons.net>.  All rights reserved.

package log4go

import (
	//"io"
	"fmt"
    "strings"
    "os"
    "github.com/rjohnsondev/raven-go/raven"
)

var (
    // number of simultaneous requests to send through to sentry
    RavenConcurrencyLevel = 32
)

// This is the standard writer that prints to standard output.
type RavenLogWriter chan *LogRecord

func NewRavenLogWriter(dsn string) RavenLogWriter {

    hostname, err := os.Hostname()
    if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to determine hostname, logging to root\n")
    }

	w := RavenLogWriter(make(chan *LogRecord, LogBufferLength))

    for x := 0; x < RavenConcurrencyLevel; x++ {
        go ravenLogWorker(w, dsn, hostname)
    }

	return w
}

func ravenLogWorker(w RavenLogWriter, dsn string, hostname string) {
    client, err := raven.NewClient(dsn, hostname)
    if err != nil {
        fmt.Fprintf(os.Stderr, "NewRavenLogWriter(%q): %s\n", dsn, err)
        return
    }

    var timestr string
    var timestrAt int64

    for rec := range w {
        if at := rec.Created.UnixNano() / 1e9; at != timestrAt {
            timestr, timestrAt = rec.Created.Format("01/02/06 15:04:05"), at
        }

        extra := make(map[string]interface{})
        extra["StackTrace"] = strings.Split(strings.Replace(string(rec.Stack), "\t", "    ",-1), "\n")
        extra["Time"] = timestr
        extra["Level"] = rec.Level
        extra["Source"] = rec.Source

        var err error = nil
        if rec.Level <= DEBUG {
            err = client.Debug(rec.Message, extra)
        } else if rec.Level <= INFO {
            err = client.Info(rec.Message, extra)
        } else if rec.Level <= WARNING {
            err = client.Warning(rec.Message, extra)
        } else if rec.Level <= ERROR {
            err = client.Error(rec.Message, extra)
        } else if rec.Level <= CRITICAL {
            err = client.Fatal(rec.Message, extra)
        }

        if err != nil {
            fmt.Printf("RavenLogWriter(%q): %s \n %s", dsn, err, rec.Message)
            return
        }
    }
}

// This is the ConsoleLogWriter's output method.  This will block if the output
// buffer is full.
func (w RavenLogWriter) LogWrite(rec *LogRecord) {
	w <- rec
}

// Close stops the logger from sending messages to standard output.  Attempts to
// send log messages to this logger after a Close have undefined behavior.
func (w RavenLogWriter) Close() {
	close(w)
}
