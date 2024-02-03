package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"internal/lib"

	"golang.org/x/exp/slog"
)

// Represents the mode value.
type queryMode int

// Queries provided as flags.
type queriesArg []string

func (q *queriesArg) String() string {
	return fmt.Sprintf("%v", &q)
}

func (q *queriesArg) Set(query string) error {
	*q = append(*q, query)
	return nil
}

// Converts to a string slice.
func (q *queriesArg) ToStrings() (q_strings []string) {
	for _, v := range *q {
		q_strings = append(q_strings, v)
	}
	return
}

// Mode constants.
const (
	MODE_QUERY   queryMode = iota + 1 // For running in 'query' mode.
	MODE_PROFILE                      // For running in 'profile' mode.
	MODE_READ                         // For running in 'read' mode.
)

var (
	attempts    int        // Number of attempts to execute the query.
	delay       int        // Delay between queries.
	displayMode int        // Result mode to display.
	filters     string     // Result filters.
	logLevel    string     // Log level.
	mode        int        // Mode to execute in.
	port        string     // Port for RPC.
	queries     queriesArg // Queries to execute.
	silent      bool       // Whether or not to be quiet.
	labels      string     // Result value labels.

	ctx                    = context.Background() // Initialize context.
	logger                 = log.Default()        // Logging system.
	logLevelStrToSlogLevel = map[string]slog.Level{
		"debug": slog.LevelDebug,
		"error": slog.LevelError,
		"info":  slog.LevelInfo,
		"warn":  slog.LevelWarn,
	} // Log levels acceptable as a flag.
)

// Parses a comma delimited argument string, returning a slice of strings if any are found, or an
// empty slice if not.
func parseCommaDelimitedArg(arg string) []string {
	if parsed := strings.Split(arg, ","); parsed[0] == "" {
		return []string{}
	} else {
		return parsed
	}
}

func main() {
	var (
		doneQueriesChan chan bool            // Channels for tracking query completion.
		pauseQueryChans map[string]chan bool // Channels for pausing queries.
	)

	defer close(doneQueriesChan)
	for _, pauseChan := range pauseQueryChans {
		defer close(pauseChan)
	}

	// Define arguments.
	flag.BoolVar(&silent, "s", false, "Don't output anything to a console.")
	flag.IntVar(&attempts, "t", 1, "Number of query executions. -1 for continuous.")
	flag.IntVar(&delay, "d", 3, "Delay between queries (seconds).")
	flag.IntVar(&displayMode, "r", int(lib.DISPLAY_MODE_RAW), "Result mode to display.")
	flag.IntVar(&mode, "m", int(MODE_QUERY), "Mode to execute in.")
	flag.StringVar(&filters, "f", "", "Results filters.")
	flag.StringVar(&logLevel, "l", "error", "Log level.")
	flag.StringVar(&port, "p", "12345", "Port for RPC.")
	flag.StringVar(&labels, "v", "", "Labels to apply to query values, separated by commas.")
	flag.Var(&queries, "q", "Query to execute. When in query mode, this is expected to be some command. When in profile mode it is expected to be PID.")
	flag.Parse()

	// Set-up logging.
	if silent || displayMode == int(lib.DISPLAY_MODE_GRAPH) {
		// Silence all output.
		logger.SetOutput(io.Discard)
	} else {
		// Set the default to be standard error--result modes may change this.
		slog.SetDefault(slog.New(slog.NewTextHandler(
			os.Stderr,
			&slog.HandlerOptions{Level: logLevelStrToSlogLevel[logLevel]},
		)))
	}

	// Execute the specified mode.
	switch {
	case mode == int(MODE_PROFILE):
		slog.Debug("Executing in profile mode.")

		doneQueriesChan, pauseQueryChans = lib.Query(
			lib.QUERY_MODE_PROFILE,
			queries,
			attempts,
			delay,
			port,
		)

		// Process mode has specific labels--ignore user provided ones.
		ctx = context.WithValue(ctx, "labels", lib.ProfileLabels)
	case mode == int(MODE_QUERY):
		slog.Debug("Executing in query mode.")

		doneQueriesChan, pauseQueryChans = lib.Query(
			lib.QUERY_MODE_COMMAND,
			queries,
			attempts,
			delay,
			port,
		)

		// Rely on user-defined labels.
		ctx = context.WithValue(ctx, "labels", parseCommaDelimitedArg(labels))
	case mode == int(MODE_READ):
		slog.Debug("Executing in read mode.")

	// FIXME Temporarily disabling read mode.
	// 	done = lib.Read(port)
	default:
		slog.Error(fmt.Sprintf("Invalid mode: %d\n", mode))
		os.Exit(1)
	}

	// Initialize remaining context.
	ctx = context.WithValue(ctx, "filters", parseCommaDelimitedArg(filters))
	ctx = context.WithValue(ctx, "queries", queries.ToStrings())

	// Execute result viewing.
	if !silent {
		lib.Results(
			ctx,
			lib.DisplayMode(displayMode),
			ctx.Value("queries").([]string)[0], // Always start with the first query.
			lib.Config{
				LogLevel: logLevel,
			},
			pauseQueryChans,
		)
	}

	<-doneQueriesChan
}
