//
// Display (TUI) related logic.

package lib

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mum4k/termdash/cell"
	"github.com/mum4k/termdash/widgets/sparkline"
	"github.com/rivo/tview"
)

// General configuration for display modes.
type DisplayConfig struct {
	HelpSize, LogsSize, ResultsSize                                          int  // Proportional size of widgets.
	OuterPaddingBottom, OuterPaddingLeft, OuterPaddingRight, OuterPaddingTop int  // Padding for the full display.
	ShowHelp, ShowLogs, ShowStatus                                           bool // Whether or not to show widgets.
	TablePadding                                                             int  // Padding for table cells in table displays.
}

// Represents the display driver.
type DisplayDriver int

// Represents the display mode.
type DisplayMode int

// Display driver constants. Each display mode uses a specific display driver.
const (
	DISPLAY_RAW      DisplayDriver = iota + 1 // Used for direct output.
	DISPLAY_TVIEW                             // Used when tview is the TUI driver.
	DISPLAY_TERMDASH                          // Used when termdash is the TUI driver.
)

// Display mode constants.
const (
	DISPLAY_MODE_RAW    DisplayMode = iota + 1 // For running in 'raw' display mode.
	DISPLAY_MODE_STREAM                        // For running in 'stream' display mode.
	DISPLAY_MODE_TABLE                         // For running in 'table' display mode.
	DISPLAY_MODE_GRAPH                         // For running in 'graph' display mode.
)

// Defaults for display configs.
const (
	DEFAULT_HELP_SIZE            = 10
	DEFAULT_LOGS_SIZE            = 15
	DEFAULT_OUTER_PADDING_BOTTOM = 5
	DEFAULT_OUTER_PADDING_LEFT   = 10
	DEFAULT_OUTER_PADDING_RIGHT  = 10
	DEFAULT_OUTER_PADDING_TOP    = 5
	DEFAULT_RESULTS_SIZE         = 75
	DEFAULT_TABLE_PADDING        = 2
)

// Misc. constants.
const (
	HELP_TEXT = "(ESC) Quit | (Space) Pause | (Tab) Next Display | (n) Next Query"
)

var (
	activeDisplayModes = []DisplayMode{
		// DISPLAY_MODE_RAW,  // It's impossible to escape raw mode, so we exclude it.
		DISPLAY_MODE_STREAM,
		DISPLAY_MODE_TABLE,
		DISPLAY_MODE_GRAPH,
	} // Display modes considered for use in the current session.
	interruptChan = make(chan bool) // Channel for interrupting displays.
)

// Starts the display. Applies contextual logic depending on the provided display driver. Expects a
// function to execute within a goroutine to update the display.
func display(driver DisplayDriver, displayUpdateFunc func()) {
	// Execute the update function.
	go displayUpdateFunc()

	switch driver {
	case DISPLAY_TVIEW:
		// Start the tview-specific display.
		err := appTview.Run()
		e(err)
	case DISPLAY_TERMDASH:
		// Start the termdash-specific display.
		// Nothing to do, yet.
	}
}

// Clean-up display logic when fully quitting.
func displayQuit() {
	close(interruptChan)
}

// Creates a default display config.
func NewDisplayConfig() *DisplayConfig {
	return &DisplayConfig{
		HelpSize:           DEFAULT_HELP_SIZE,
		LogsSize:           DEFAULT_LOGS_SIZE,
		OuterPaddingBottom: DEFAULT_OUTER_PADDING_BOTTOM,
		OuterPaddingLeft:   DEFAULT_OUTER_PADDING_LEFT,
		OuterPaddingRight:  DEFAULT_OUTER_PADDING_RIGHT,
		OuterPaddingTop:    DEFAULT_OUTER_PADDING_TOP,
		ResultsSize:        DEFAULT_RESULTS_SIZE,
		ShowHelp:           true,
		ShowLogs:           false,
		ShowStatus:         true,
		TablePadding:       DEFAULT_TABLE_PADDING,
	}
}

// Presents raw output.
func RawDisplay(query string) {
	var (
		reader = readerIndexes[query] // Reader index for the query.
	)

	// Wait for the first result to appear to synchronize storage.
	GetResultWait(query)
	reader.Dec()

	// Load existing results.
	for _, result := range store.GetToIndex(query, reader) {
		fmt.Println(result)
	}

	// Load new results.
	for {
		fmt.Println(GetResult(query))
	}
}

// Update the results pane with new results as they are generated.
func StreamDisplay(query string, filters, labels []string, displayConfig *DisplayConfig) {
	var (
		reader = readerIndexes[query] // Reader index for the query.
	)

	// ait for the first result to appear to synchronize storage.
	GetResultWait(query)
	reader.Dec()

	// Initialize the display.
	widgets := initDisplayTviewText(query, filters, labels, displayConfig)

	// Start the display.
	display(
		DISPLAY_TVIEW,
		func() {
			// Print labels as the first line.
			appTview.QueueUpdateDraw(func() {
				fmt.Fprintln(widgets.resultsWidget.(*tview.TextView), labels)
			})

			// Print all previous results.
			for _, result := range store.GetToIndex(query, reader) {
				fmt.Fprintln(widgets.resultsWidget.(*tview.TextView), result.Value)
			}

			// Print results.
			for {
				// Listen for an interrupt to stop result consumption for some display change.
				select {
				case <-interruptChan:
					// We've received an interrupt.
					return
				case <-pauseDisplayChan:
					// We've received a pause and need to wait for an unpause.
					<-pauseDisplayChan
				default:
					// We can display the next result.
					fmt.Fprintln(widgets.resultsWidget.(*tview.TextView), (GetResult(query)).Value)
				}
			}
		},
	)
}

// Creates a table of results for the results pane.
func TableDisplay(query string, filters, labels []string, displayConfig *DisplayConfig) {
	var (
		widgets tviewWidgets // Widgets produced by tview.

		reader           = readerIndexes[query]                            // Reader index for the query.
		tableCellPadding = strings.Repeat(" ", displayConfig.TablePadding) // Padding to add to table cell content.
		valueIndexes     = []int{}                                         // Indexes of the result values to add to the table.
	)

	// Wait for the first result to appear to synchronize storage.
	GetResultWait(query)
	reader.Dec()

	// Initialize the display.
	widgets = initDisplayTviewTable(query, filters, labels, displayConfig)

	// Start the display.
	display(
		DISPLAY_TVIEW,
		func() {
			var (
				nextCellContent string // Next cell to add to the table.

				i = 0 // Used to determine the next row index.
			)

			// Determine the value indexes to populate into the graph. If no filter is provided, the index
			// is assumed to be zero.
			if len(filters) > 0 {
				for _, filter := range filters {
					valueIndexes = append(valueIndexes, store.GetValueIndex(query, filter))
				}
			}

			appTview.QueueUpdateDraw(func() {
				// Row to contain the labels.
				headerRow := widgets.resultsWidget.(*tview.Table).InsertRow(i)

				for j, label := range FilterSlice(labels, valueIndexes) {
					headerRow.SetCellSimple(i, j, tableCellPadding+label+tableCellPadding)
				}
			})
			i += 1

			// Print all previous results.
			for _, result := range store.GetToIndex(query, reader) {
				appTview.QueueUpdateDraw(func() {
					var (
						row = widgets.resultsWidget.(*tview.Table).InsertRow(i) // Row to contain the result.
					)

					for j, value := range FilterSlice(result.Values, valueIndexes) {
						// Extrapolate the field types in order to print them out.
						switch value.(type) {
						case int64:
							nextCellContent = strconv.FormatInt(value.(int64), 10)
						case float64:
							nextCellContent = strconv.FormatFloat(value.(float64), 'f', -1, 64)
						default:
							nextCellContent = value.(string)
						}
						row.SetCellSimple(i, j, tableCellPadding+nextCellContent+tableCellPadding)
					}
				})
				i += 1
			}

			// Print results.
			for {
				// Listen for an interrupt to stop result consumption for some display change.
				select {
				case <-interruptChan:
					// We've received an interrupt.
					return
				case <-pauseDisplayChan:
					// We've received a pause and need to wait for an unpause.
					<-pauseDisplayChan
				default:
					// We can display the next result.
					appTview.QueueUpdateDraw(func() {
						var (
							row = widgets.resultsWidget.(*tview.Table).InsertRow(i) // Row to contain the result.
						)

						for j, value := range FilterSlice((GetResult(query)).Values, valueIndexes) {
							// Extrapolate the field types in order to print them out.
							switch value.(type) {
							case int64:
								nextCellContent = strconv.FormatInt(value.(int64), 10)
							case float64:
								nextCellContent = strconv.FormatFloat(value.(float64), 'f', -1, 64)
							default:
								nextCellContent = value.(string)
							}
							row.SetCellSimple(i, j, tableCellPadding+nextCellContent+tableCellPadding)
						}
					})
					i += 1
				}
			}
		},
	)
}

// Creates a graph of results for the results pane.
func GraphDisplay(query string, filters, labels []string, displayConfig *DisplayConfig) {
	var (
		err error // General error holder.

		reader     = readerIndexes[query] // Reader index for the query.
		valueIndex = 0                    // Index of the result value to graph.
		widgets    = termdashWidgets{}    // Widgets for displaying.
	)

	// Wait for the first result to appear to synchronize storage.
	GetResultWait(query)
	reader.Dec()

	// Determine the values to populate into the graph. If none is provided, the first value is taken.
	if len(filters) > 0 {
		valueIndex = store.GetValueIndex(query, filters[0])
	}

	// Initialize the results view.
	//
	// XXX This should probably moved into `display_termdash.go` once termdash is managing more types
	// of result displays.
	widgets.resultsWidget, err = sparkline.New(
		sparkline.Label(labels[valueIndex]),
		sparkline.Color(cell.ColorGreen),
	)
	e(err)

	// Start the display.
	display(
		DISPLAY_TERMDASH,
		func() {
			// Print all previous results.
			for _, result := range store.GetToIndex(query, reader) {
				// We can display the next result.
				value := result.Values.Get(valueIndex)

				switch value.(type) {
				case int64:
					widgets.resultsWidget.(*sparkline.SparkLine).Add([]int{int(value.(int64))})
				case float64:
					widgets.resultsWidget.(*sparkline.SparkLine).Add([]int{int(value.(float64))})
				}
			}

			for {
				// Listen for an interrupt to stop result consumption for some display change.
				select {
				case <-interruptChan:
					// We've received an interrupt.
					return
				case <-pauseDisplayChan:
					// We've received a pause and need to wait for an unpause.
					<-pauseDisplayChan
				default:
					// We can display the next result.
					value := (GetResult(query)).Values.Get(valueIndex)

					switch value.(type) {
					case int64:
						widgets.resultsWidget.(*sparkline.SparkLine).Add([]int{int(value.(int64))})
					case float64:
						widgets.resultsWidget.(*sparkline.SparkLine).Add([]int{int(value.(float64))})
					}
				}
			}
		},
	)

	// Initialize the display. This must happen after the display function is invoked, otherwise data
	// will never appear.
	initDisplayTermdash(widgets, query, filters, labels, displayConfig)
}
