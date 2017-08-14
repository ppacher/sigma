package trigger

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/Knetic/govaluate"
	"github.com/homebot/sigma"
	"github.com/yalp/jsonpath"
)

func getTimePart(t interface{}, what string) (interface{}, error) {
	var tm time.Time

	switch v := t.(type) {
	case int:
		tm = time.Unix(int64(v), 0)
	case int64:
		tm = time.Unix(v, 0)
	case float64:
		tm = time.Unix(int64(v), 0)
	case float32:
		tm = time.Unix(int64(v), 0)
	case string:
		var err error
		tm, err = time.Parse(v, time.RFC3339)
		if err != nil {
			return nil, err
		}
	case time.Time:
		tm = v
	default:
		return nil, fmt.Errorf("invalid argument: %#v (%s)", t, reflect.TypeOf(t))
	}

	switch what {
	case "hour":
		return float64(tm.Hour()), nil
	case "minute":
		return float64(tm.Minute()), nil
	case "second":
		return float64(tm.Second()), nil
	case "day":
		return float64(tm.Day()), nil
	case "weekday":
		return tm.Weekday().String(), nil
	default:
		return nil, errors.New("unknown date format")
	}
}

func buildTimeFunc(what string) func(...interface{}) (interface{}, error) {
	return func(args ...interface{}) (interface{}, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("usage: %s(date)", what)
		}
		return getTimePart(args[0], what)
	}
}

// Evaluate evaluates the condtion on event
func Evaluate(condtion string, event sigma.Event) (bool, error) {
	if condtion == "" {
		return true, nil
	}

	functions := map[string]govaluate.ExpressionFunction{
		"jsonpath": func(args ...interface{}) (interface{}, error) {
			if len(args) != 2 {
				return nil, errors.New("usage: jsonpath(blob, path)")
			}

			blob, ok := args[0].(string)
			if !ok {
				return nil, errors.New("`blob` mustbe string")
			}

			path, ok := args[1].(string)
			if !ok {
				return nil, errors.New("`path` must be string")
			}

			var res interface{}
			if err := json.Unmarshal([]byte(blob), &res); err != nil {
				return nil, err
			}

			return jsonpath.Read(res, path)
		},
		"contains": func(args ...interface{}) (interface{}, error) {
			if len(args) != 2 {
				return nil, errors.New("usage: contains(string, substring)")
			}

			s1, ok1 := args[0].(string)
			s2, ok2 := args[1].(string)

			if !ok1 || !ok2 {
				return nil, errors.New("string and substring must be strings")
			}

			return strings.Contains(s1, s2), nil
		},
		"second":  buildTimeFunc("second"),
		"minute":  buildTimeFunc("minute"),
		"hour":    buildTimeFunc("hour"),
		"day":     buildTimeFunc("day"),
		"weekday": buildTimeFunc("weekday"),
	}

	expr, err := govaluate.NewEvaluableExpressionWithFunctions(condtion, functions)
	if err != nil {
		return false, err
	}

	parameters := map[string]interface{}{
		"type":    event.Type(),
		"payload": string(event.Payload()),
	}

	res, err := expr.Evaluate(parameters)
	if err != nil {
		return false, err
	}

	b, ok := res.(bool)
	if ok {
		return b, nil
	}

	return false, fmt.Errorf("unsupported return value: %#v (%s)", res, reflect.TypeOf(res))
}
