package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/araddon/dateparse"
	"github.com/fatih/color"
	"github.com/zefhemel/ax/pkg/backend/common"
	"github.com/zefhemel/ax/pkg/complete"
	"github.com/zefhemel/ax/pkg/config"
	yaml "gopkg.in/yaml.v2"
)

var (
	queryBefore     = queryCommand.Flag("before", "Results from before").String()
	queryAfter      = queryCommand.Flag("after", "Results from after").String()
	queryMaxResults = queryCommand.Flag("results", "Maximum number of results").Short('n').Default("200").Int()
	querySelect     = queryCommand.Flag("select", "Fields to select").Short('s').HintAction(selectHintAction).Strings()
	queryWhere      = queryCommand.Flag("where", "Add a filter").Short('w').HintAction(whereHintAction).Strings()
	//querySortDesc     = queryCommand.Flag("desc", "Sort results reverse-chronologically").Default("false").Bool()
	queryOutputFormat = queryCommand.Flag("output", "Output format: text|json|yaml").Short('o').Default("text").Enum("text", "yaml", "json", "pretty-json")
	queryFollow       = queryCommand.Flag("follow", "Follow log in quasi-realtime, similar to tail -f").Short('f').Default("false").Bool()
	queryString       = queryCommand.Arg("query", "Query string").Default("").Strings()
)

func whereHintAction() []string {
	rc := config.BuildConfig()
	resultList := make([]string, 0, 20)
	for attrName, _ := range complete.GetCompletions(rc) {
		resultList = append(resultList, fmt.Sprintf("%s=", attrName))
	}
	return resultList
}

func selectHintAction() []string {
	rc := config.BuildConfig()
	resultList := make([]string, 0, 20)
	for attrName, _ := range complete.GetCompletions(rc) {
		resultList = append(resultList, attrName)
	}
	return resultList
}

func buildFilters(wheres []string) []common.QueryFilter {
	filters := make([]common.QueryFilter, 0, len(wheres))
	for _, whereClause := range wheres {
		pieces := strings.SplitN(whereClause, "=", 2)
		if len(pieces) != 2 {
			fmt.Println("Invalid where clause", whereClause)
			os.Exit(1)
		}
		filters = append(filters, common.QueryFilter{
			FieldName: pieces[0],
			Value:     pieces[1],
		})
	}
	return filters
}

func queryMain(rc config.RuntimeConfig, client common.Client) {
	var before *time.Time
	var after *time.Time
	if *queryAfter != "" {
		var err error
		afterTime, err := dateparse.ParseLocal(*queryAfter)
		if err != nil {
			fmt.Println("Could parse after date:", *queryAfter)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Parsed --after as %s", afterTime.Format(common.TimeFormat))
		after = &afterTime
	}
	if *queryBefore != "" {
		var err error
		beforeTime, err := dateparse.ParseLocal(*queryBefore)
		if err != nil {
			fmt.Println("Could parse before date:", *queryBefore)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Parsed --before as %s", beforeTime.Format(common.TimeFormat))
		before = &beforeTime
	}

	for message := range complete.GatherCompletionInfo(rc, client.Query(common.Query{
		QueryString:  strings.Join(*queryString, " "),
		Before:       before,
		After:        after,
		Filters:      buildFilters(*queryWhere),
		MaxResults:   *queryMaxResults,
		SelectFields: *querySelect,
		Follow:       *queryFollow,
	})) {
		printMessage(message, *queryOutputFormat)
	}

}

func printMessage(message common.LogMessage, queryOutputFormat string) {
	switch queryOutputFormat {
	case "text":
		ts := message.Timestamp.Format(common.TimeFormat)
		fmt.Printf("[%s] ", color.MagentaString(ts))
		if msg, ok := message.Attributes["message"].(string); ok {
			messageColor := color.New(color.Bold)
			fmt.Printf("%s ", messageColor.Sprint(msg))
		}
		for key, value := range message.Attributes {
			if key == "message" {
				continue
			}
			fmt.Printf("%s=%s ", color.CyanString(key), common.MustJsonEncode(value))
		}
		fmt.Println()
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		err := encoder.Encode(message.Map())
		if err != nil {
			fmt.Println("Error JSON encoding")
		}
	case "pretty-json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		err := encoder.Encode(message.Map())
		if err != nil {
			fmt.Println("Error JSON encoding")
		}
	case "yaml":
		buf, err := yaml.Marshal(message.Map())
		if err != nil {
			fmt.Println("Error YAML encoding")
		}
		fmt.Printf("---\n%s", string(buf))
	}
}
