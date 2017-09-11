package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/araddon/dateparse"
	"github.com/egnyte/ax/pkg/backend/common"
	"github.com/egnyte/ax/pkg/complete"
	"github.com/egnyte/ax/pkg/config"
	"github.com/fatih/color"
	"github.com/zefhemel/kingpin"
	yaml "gopkg.in/yaml.v2"
)

func addQueryFlags(cmd *kingpin.CmdClause) *common.QuerySelectors {
	flags := &common.QuerySelectors{}
	cmd.Flag("before", "Results from before").StringVar(&flags.Before)
	cmd.Flag("after", "Results from after").StringVar(&flags.After)
	cmd.Flag("select", "Fields to select").Short('s').HintAction(selectHintAction).StringsVar(&flags.Select)
	cmd.Flag("where", "Add a filter").Short('w').HintAction(whereHintAction).StringsVar(&flags.Where)
	cmd.Arg("query", "Query string").Default("").StringsVar(&flags.QueryString)
	return flags
}

var (
	queryFlags            = addQueryFlags(queryCommand)
	queryFlagMaxResults   int
	queryFlagOutputFormat string
	queryFlagFollow       bool
)

func init() {
	queryCommand.Flag("results", "Maximum number of results").Short('n').Default("200").IntVar(&queryFlagMaxResults)
	queryCommand.Flag("output", "Output format: text|json|yaml").Short('o').Default("text").EnumVar(&queryFlagOutputFormat, "text", "yaml", "json", "pretty-json")
	queryCommand.Flag("follow", "Follow log in quasi-realtime, similar to tail -f").Short('f').Default("false").BoolVar(&queryFlagFollow)
}

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

var filterRegex = regexp.MustCompile(`([^!=<>]+)\s*(=|!=)\s*(.*)`)

func buildFilters(wheres []string) []common.QueryFilter {
	filters := make([]common.QueryFilter, 0, len(wheres))
	for _, whereClause := range wheres {
		//pieces := strings.SplitN(whereClause, "=", 2)
		matches := filterRegex.FindAllStringSubmatch(whereClause, -1)
		if len(matches) != 1 {
			fmt.Println("Invalid where clause", whereClause)
			os.Exit(1)
		}
		filters = append(filters, common.QueryFilter{
			FieldName: matches[0][1],
			Operator:  matches[0][2],
			Value:     matches[0][3],
		})
	}
	return filters
}

func querySelectorsToQuery(flags *common.QuerySelectors) common.Query {
	var before *time.Time
	var after *time.Time
	if flags.After != "" {
		var err error
		afterTime, err := dateparse.ParseLocal(flags.After)
		if err != nil {
			fmt.Println("Could parse after date:", flags.After)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Parsed --after as %s", afterTime.Format(common.TimeFormat))
		after = &afterTime
	}
	if flags.Before != "" {
		var err error
		beforeTime, err := dateparse.ParseLocal(flags.Before)
		if err != nil {
			fmt.Println("Could parse before date:", flags.Before)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Parsed --before as %s", beforeTime.Format(common.TimeFormat))
		before = &beforeTime
	}

	return common.Query{
		QueryString:  strings.Join(flags.QueryString, " "),
		Before:       before,
		After:        after,
		Filters:      buildFilters(flags.Where),
		SelectFields: flags.Select,
	}
}

func queryMain(rc config.RuntimeConfig, client common.Client) {
	query := querySelectorsToQuery(queryFlags)
	query.MaxResults = queryFlagMaxResults
	query.Follow = queryFlagFollow
	for message := range complete.GatherCompletionInfo(rc, client.Query(query)) {
		printMessage(message, queryFlagOutputFormat)
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
			if key == "message" || value == nil {
				continue
			}
			fmt.Printf("%s=%+v ", color.CyanString(key), value)
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
