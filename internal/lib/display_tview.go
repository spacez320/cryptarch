//
// Display management for modes using Tview.

package lib

import (
	"fmt"
	"os"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"golang.org/x/exp/slog"
)

// Display init function specific to table results.
func initDisplayTviewTable(helpText string) (resultsView *tview.Table, helpView, logsView *tview.TextView) {
	// Initialize the results view.
	resultsView = tview.NewTable()
	resultsView.SetBorders(true).SetDoneFunc(
		func(key tcell.Key) {
			switch key {
			case tcell.KeyEscape:
				// When a user presses Esc, close the application.
				app.Stop()
				os.Exit(0)
			}
		},
	)
	resultsView.SetBorder(true).SetTitle("Results")

	helpView, logsView = initDisplayTview(resultsView, helpText)

	return
}

// Display init function specific to text results.
func initDisplayTviewText(helpText string) (resultsView, helpView, logsView *tview.TextView) {
	// Initialize the results view.
	resultsView = tview.NewTextView()
	resultsView.SetChangedFunc(
		func() {
			app.Draw()
		}).SetDoneFunc(
		func(key tcell.Key) {
			switch key {
			case tcell.KeyEscape:
				// When a user presses Esc, close the application.
				app.Stop()
				os.Exit(0)
			}
		},
	)
	resultsView.SetBorder(true).SetTitle("Results")

	helpView, logsView = initDisplayTview(resultsView, helpText)

	return
}

// Sets-up the tview flex box, which defines the overall layout. Meant to
// encapsulate the common things needed regardless of what from the results
// view takes (assuming it fits into flex box).
//
// Note that the app needs to be run separately from initialization in the
// coroutine display function. Note also that direct manipulation of the tview
// Primitives as subclasses (like tview.Box) needs to happen outside this
// function, as well.
func initDisplayTview(resultsView tview.Primitive, helpText string) (helpView, logsView *tview.TextView) {
	var (
		flexBox = tview.NewFlex()
	)

	helpView = tview.NewTextView()
	logsView = tview.NewTextView()

	// Set-up the layout and apply views.
	flexBox = flexBox.SetDirection(tview.FlexRow).
		AddItem(resultsView, 0, RESULTS_SIZE, false).
		AddItem(helpView, 0, HELP_SIZE, false).
		AddItem(logsView, 0, LOGS_SIZE, false)
	flexBox.SetBorderPadding(
		OUTER_PADDING_TOP,
		OUTER_PADDING_BOTTOM,
		OUTER_PADDING_LEFT,
		OUTER_PADDING_RIGHT,
	)
	app.SetRoot(flexBox, true).SetFocus(resultsView)

	// Initialize the help view.
	helpView.SetBorder(true).SetTitle("Help")
	fmt.Fprintln(helpView, helpText)

	// Initialize the logs view.
	logsView.SetBorder(true).SetTitle("Logs")
	slog.SetDefault(slog.New(slog.NewTextHandler(
		logsView,
		&slog.HandlerOptions{Level: config.SlogLogLevel()},
	)))

	return helpView, logsView
}
