//
// Manages individual or series of results for storage.

package storage

import (
	"fmt"
	_ "log/slog"
	"reflect"
	"strconv"
	"time"

	"golang.org/x/exp/slices"
)

// Tokenized result value.
type Values []interface{}

// Retrieve an indexed token.
func (v Values) Get(index int) (result interface{}) {
	// Return an empty value if we didn't find an indexed value. This can happen if empty results are
	// stored.
	if len(v) > index {
		result = v[index]
	}

	return
}

// Individual result.
type Result struct {
	Time   time.Time // Time the result was created.
	Value  string    // Raw value of the result.
	Values Values    // Tokenized value of the result.
}

// Determines whether this is an empty result.
func (r *Result) IsEmpty() bool {
	return reflect.DeepEqual(*r, Result{})
}

// Determines whether this result contains empty values.
func (r *Result) IsEmptyValues() bool {
	return (*r).Value == "" && len((*r).Values) == 0
}

// Returns a map of values keyed to their labels.
func (r *Result) Map(labels []string) map[string]interface{} {
	resultMap := make(map[string]interface{}, len(r.Values))
	for i, value := range r.Values {
		resultMap[labels[i]] = value
	}

	return resultMap
}

// Collection of results.
type Results struct {
	// Meta field for result values acting as a name, corresponding by index. In the event that no
	// explicit labels are defined, the indexes are the labels.
	Labels []string
	// Stored results.
	Results []Result
}

// Get a result based on a timestamp.
func (r *Results) get(time time.Time) Result {
	for _, result := range (*r).Results {
		if result.Time.Compare(time) == 0 {
			// We found a result to return.
			return result
		}
	}

	// Return an empty result if nothing was discovered.
	return Result{}
}

// Gets results based on a start and end timestamp.
func (r *Results) getRange(startTime time.Time, endTime time.Time) (found []Result) {
	for _, result := range (*r).Results {
		if result.Time.Compare(startTime) >= 0 {
			if result.Time.Compare(endTime) > 0 {
				// Break out of the loop if we've exhausted the upper bounds of the range.
				break
			} else {
				found = append(found, result)
			}
		}
	}

	return
}

// Given a filter, return the corresponding value index.
func (r *Results) getValueIndex(filter string) int {
	return slices.Index((*r).Labels, filter)
}

// Put a new compound result.
func (r *Results) put(value string, values ...interface{}) Result {
	next := Result{
		Time:   time.Now(),
		Value:  value,
		Values: values,
	}

	(*r).Results = append((*r).Results, next)

	return next
}

// Show all currently stored results.
func (r *Results) show() {
	for _, result := range (*r).Results {
		fmt.Printf("Label: %v, Time: %v, Value: %v, Values: %v\n",
			(*r).Labels, result.Time, result.Value, result.Values)
	}
}

// Creates new results.
func newResults(size int) (results Results) {
	// Iniitialize labels.
	results.Labels = make([]string, size)
	for i := range results.Labels {
		results.Labels[i] = strconv.Itoa(i)
	}

	return
}
