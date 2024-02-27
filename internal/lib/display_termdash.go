//
// Display management for modes using Termdash.

package lib

import (
	"context"
	"fmt"

	"github.com/mum4k/termdash"
	"github.com/mum4k/termdash/container"
	"github.com/mum4k/termdash/keyboard"
	"github.com/mum4k/termdash/linestyle"
	"github.com/mum4k/termdash/terminal/tcell"
	"github.com/mum4k/termdash/terminal/terminalapi"
	"github.com/mum4k/termdash/widgetapi"
	"github.com/mum4k/termdash/widgets/text"
	"golang.org/x/exp/slog"
)

// Used to provide an io.Writer implementation of termdash text widgets.
type termdashTextWriter struct {
	text text.Text
}

// Implements io.Writer.
func (t *termdashTextWriter) Write(p []byte) (n int, err error) {
	t.text.Write(string(p))
	return len(p), nil
}

// Used to supply optional widgets to Termdash initialization.
type termdashWidgets struct {
	filterWidget, helpWidget, labelWidget, logsWidget, queryWidget *text.Text
	resultsWidget                                                  widgetapi.Widget
}

var (
	appTermdash *tcell.Terminal    // Termdash display.
	cancel      context.CancelFunc // Cancel function for the termdash display.
)

// Function to call on keyboard events.
func keyboardTermdashHandler(key *terminalapi.Keyboard) {
	switch key.Key {
	case keyboard.KeyEsc:
		// Escape quits the program.
		slog.Debug("Quitting.")

		currentCtx = context.WithValue(currentCtx, "quit", true)
		cancel()
		appTermdash.Close()
	case keyboard.KeyTab:
		// Tab switches display modes.
		slog.Debug("Switching display mode.")

		interruptChan <- true
		currentCtx = context.WithValue(currentCtx, "advanceDisplayMode", true)
		cancel()
		appTermdash.Close()
	case 'n':
		// 'n' switches queries.
		slog.Debug("Switching query.")

		interruptChan <- true
		currentCtx = context.WithValue(currentCtx, "advanceQuery", true)
		cancel()
		appTermdash.Close()
	case ' ':
		// Space pauses.
		slog.Debug("Pausing.")

		pauseDisplayChan <- true
		pauseQueryChans[currentCtx.Value("query").(string)] <- true
	}
}

// Error management for termdash.
func errorTermdashHandler(e error) {
	// If we hit an error from termdash, just log it and try to continue. Cases of errors seen so far
	// make sense to ignore:
	//
	// - Unimplemented key-strokes.
	slog.Error(e.Error())
}

// Sets-up the termdash container, which defines the overall layout, and begins running the display.
// func initDisplayTermdash(resultsWidget, helpWidget, logsWidget widgetapi.Widget) {
func initDisplayTermdash(
	widgets termdashWidgets,
	query string,
	filters, labels []string,
	showHelp, showLogs bool,
) {
	var (
		ctx              context.Context      // Termdash specific context.
		err              error                // General error holder.
		logsWidgetWriter termdashTextWriter   // Writer implementation for logs.
		topWidgets       container.Option     // Status and result widgets.
		widgetContainer  *container.Container // Wrapper for widgets.
	)
	widgets.filterWidget, err = text.New()
	e(err)
	widgets.labelWidget, err = text.New()
	e(err)
	widgets.queryWidget, err = text.New()
	e(err)

	// Instantiate optional displays.
	if showHelp {
		widgets.helpWidget, err = text.New()
		e(err)
		widgets.helpWidget.Write(HELP_TEXT)
	}
	if showLogs {
		widgets.logsWidget, err = text.New()
		e(err)
	}

	// Set-up the context and enable it to close on key-press.
	ctx, cancel = context.WithCancel(context.Background())

	// Set-up the layout.
	appTermdash, err = tcell.New()
	e(err)

	// Set-up the status widgets with results.
	topWidgets = container.SplitHorizontal(
		container.Top(
			container.SplitVertical(
				container.Left(
					container.Border(linestyle.Light),
					container.BorderTitle("Query"),
					container.BorderTitleAlignCenter(),
					container.PlaceWidget(widgets.queryWidget),
				),
				container.Right(
					container.SplitVertical(
						container.Left(
							container.Border(linestyle.Light),
							container.BorderTitle("Labels"),
							container.BorderTitleAlignCenter(),
							container.PlaceWidget(widgets.labelWidget),
						),
						container.Right(
							container.Border(linestyle.Light),
							container.BorderTitle("Filters"),
							container.BorderTitleAlignCenter(),
							container.PlaceWidget(widgets.labelWidget),
						),
					),
				),
				container.SplitPercent(33),
			),
		),
		container.Bottom(
			container.Border(linestyle.Light),
			container.BorderTitle("Results"),
			container.BorderTitleAlignCenter(),
			container.PlaceWidget(widgets.resultsWidget),
		),
		container.SplitOption(container.SplitFixed(3)),
	)

	if widgets.helpWidget != nil && widgets.logsWidget != nil {
		// All widgets enabled.
		widgetContainer, err = container.New(
			appTermdash,
			container.PaddingBottom(OUTER_PADDING_BOTTOM),
			container.PaddingLeft(OUTER_PADDING_LEFT),
			container.PaddingTop(OUTER_PADDING_TOP),
			container.PaddingRight(OUTER_PADDING_RIGHT),
			container.SplitHorizontal(
				container.Top(topWidgets),
				container.Bottom(
					container.SplitHorizontal(
						container.Top(
							container.Border(linestyle.Light),
							container.BorderTitle("Help"),
							container.BorderTitleAlignCenter(),
							container.PlaceWidget(widgets.helpWidget),
						),
						container.Bottom(
							container.Border(linestyle.Light),
							container.BorderTitle("Logs"),
							container.BorderTitleAlignCenter(),
							container.PlaceWidget(widgets.logsWidget),
						),
						container.SplitOption(container.SplitFixed(3)),
					),
				),
				// XXX The +5 is to try to match tview's proportions.
				container.SplitOption(container.SplitPercent(RESULTS_SIZE+5)),
			),
		)
	} else if widgets.helpWidget != nil {
		// We have just the help widget enabled.
		widgetContainer, err = container.New(
			appTermdash,
			container.PaddingBottom(OUTER_PADDING_BOTTOM),
			container.PaddingLeft(OUTER_PADDING_LEFT),
			container.PaddingTop(OUTER_PADDING_TOP),
			container.PaddingRight(OUTER_PADDING_RIGHT),
			container.SplitHorizontal(
				container.Top(topWidgets),
				container.Bottom(
					container.Border(linestyle.Light),
					container.BorderTitle("Help"),
					container.BorderTitleAlignCenter(),
					container.PlaceWidget(widgets.helpWidget),
				),
				// TODO This is a heuristic to make the Help pane small because termdash does not allow
				// sizing on latter elements.
				//
				// See: https://github.com/mum4k/termdash/issues/292
				container.SplitOption(container.SplitPercent(97)),
			),
		)
	} else if widgets.logsWidget != nil {
		// We have just the logs widget enabled. We also need to point logs to it.
		logsWidgetWriter = termdashTextWriter{text: *widgets.logsWidget}
		slog.SetDefault(slog.New(slog.NewTextHandler(
			&logsWidgetWriter,
			&slog.HandlerOptions{Level: config.SlogLogLevel()},
		)))

		widgetContainer, err = container.New(
			appTermdash,
			container.PaddingBottom(OUTER_PADDING_BOTTOM),
			container.PaddingLeft(OUTER_PADDING_LEFT),
			container.PaddingTop(OUTER_PADDING_TOP),
			container.PaddingRight(OUTER_PADDING_RIGHT),
			container.SplitHorizontal(
				container.Top(topWidgets),
				container.Bottom(
					container.Border(linestyle.Light),
					container.BorderTitle("Logs"),
					container.BorderTitleAlignCenter(),
					container.PlaceWidget(&logsWidgetWriter.text),
				),
				// XXX The -1 is to try to match tview's proportions.
				container.SplitOption(container.SplitPercent(RESULTS_SIZE+HELP_SIZE-1)),
			),
		)
	} else {
		// Just the results pane.
		widgetContainer, err = container.New(
			appTermdash,
			container.PaddingBottom(OUTER_PADDING_BOTTOM),
			container.PaddingLeft(OUTER_PADDING_LEFT),
			container.PaddingTop(OUTER_PADDING_TOP),
			container.PaddingRight(OUTER_PADDING_RIGHT),
			topWidgets,
		)
	}
	e(err)

	// Initialize the top-line status widgets.
	widgets.queryWidget.Write(query)
	widgets.filterWidget.Write(fmt.Sprintf("%v", filters))
	widgets.labelWidget.Write(fmt.Sprintf("%v", labels))

	// Run the display.
	termdash.Run(
		ctx,
		appTermdash,
		widgetContainer,
		termdash.ErrorHandler(errorTermdashHandler),
		termdash.KeyboardSubscriber(keyboardTermdashHandler),
	)
}
