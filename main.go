package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"golang.org/x/exp/slog"
)

////////////////////////////////////////////////////////////////////////////////
//
// Types
//
////////////////////////////////////////////////////////////////////////////////

// Represents the mode value.
type mode_ int

// Queries provided as flags.
type queries_ []string

func (q *queries_) String() string {
	return fmt.Sprintf("%v", &q)
}

func (q *queries_) Set(query string) error {
	*q = append(*q, query)
	return nil
}

// Represents the result mode value.
type resultMode_ int

////////////////////////////////////////////////////////////////////////////////
//
// Variables
//
////////////////////////////////////////////////////////////////////////////////

const (
	MODE_QUERY      mode_       = iota // For running in 'query' mode.
	MODE_READ                          // For running in 'read' mode.
	RESULT_MODE_RAW resultMode_ = iota // For running in 'raw' result mode.
)

var (
	attempts   int      // Number of attempts to execute the query.
	delay      int      // Delay between queries.
	logLevel   string   // Log level.
	mode       int      // Mode to execute in.
	port       string   // Port for RPC.
	queries    queries_ // Queries to execute.
	resultMode int      // Result mode to display.
	silent     bool     // Whether or not to be quiet.

	logger                 = log.Default() // Logging system.
	logLevelStrToSlogLevel = map[string]slog.Level{
		"debug": slog.LevelDebug,
		"info":  slog.LevelInfo,
		"warn":  slog.LevelWarn,
		"error": slog.LevelError,
	} // Log levels acceptable as a flag.
)

////////////////////////////////////////////////////////////////////////////////
//
// Private
//
////////////////////////////////////////////////////////////////////////////////

// General error manager.
func e(err error) {
	if err != nil {
		slog.Error(err.Error())
	}
}

func main() {
	// Define arguments.

	flag.BoolVar(&silent, "s", false, "Don't output anything to a console.")
	flag.IntVar(&attempts, "t", 1, "Number of query executions. -1 for continuous.")
	flag.IntVar(&delay, "d", 3, "Delay between queries (seconds).")
	flag.IntVar(&mode, "m", int(MODE_QUERY), "Mode to execute in.")
	flag.StringVar(&logLevel, "l", "error", "Log level.")
	flag.IntVar(&resultMode, "r", int(RESULT_MODE_RAW), "Result mode to display.")
	flag.StringVar(&port, "p", "12345", "Port for RPC.")
	flag.Var(&queries, "q", "Query to execute.")
	flag.Parse()

	// Set-up logging.

	if silent {
		// Silence all output.
		logger.SetOutput(ioutil.Discard)
	} else {
		slog.SetDefault(slog.New(slog.NewTextHandler(
			os.Stderr,
			&slog.HandlerOptions{Level: logLevelStrToSlogLevel[logLevel]},
		)))
	}

	// Execute the specified mode.

	done := make(chan int)
	switch {
	case mode == int(MODE_QUERY):
		slog.Debug("Executing in query mode.")
		done = Query()
		RawResults()
	case mode == int(MODE_READ):
		slog.Debug("Executing in read mode.")
		done = Read()
	default:
		slog.Error(fmt.Sprintf("Invalid mode: %v", mode))
	}
	<-done
}
