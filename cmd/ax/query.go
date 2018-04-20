package main

import (
	"context"
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
	cmd.Flag("one-of", "Add a membership filter (FIELD_NAME:FIELD_VALUE)").HintAction(oneOfHintAction).StringsVar(&flags.OneOf)
	cmd.Flag("not-one-of", "Add an inverse membership filter (FIELD_NAME:FIELD_VALUE)").HintAction(oneOfHintAction).StringsVar(&flags.NotOneOf)
	cmd.Flag("exists", "Add a field existence filter").HintAction(existenceHintAction).StringsVar(&flags.Exists)
	cmd.Flag("not-exists", "Add an inverse field existence filter").HintAction(existenceHintAction).StringsVar(&flags.NotExists)
	cmd.Flag("uniq", "Unique log messages only").Default("false").BoolVar(&flags.Unique)
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
	queryCommand.Flag("results", "Maximum number of results").Short('n').Default("50").IntVar(&queryFlagMaxResults)
	queryCommand.Flag("output", "Output format: text|json|yaml").Short('o').Default("text").EnumVar(&queryFlagOutputFormat, "text", "yaml", "json", "pretty-json")
	queryCommand.Flag("follow", "Follow log in quasi-realtime, similar to tail -f").Short('f').Default("false").BoolVar(&queryFlagFollow)
}

func commonHintAction(suffix string) []string {
	rc := config.BuildConfig()
	resultList := make([]string, 0, 20)
	for attrName := range complete.GetCompletions(rc) {
		resultList = append(resultList, fmt.Sprintf("%s%s", attrName, suffix))
	}
	return resultList
}

func whereHintAction() []string {
	return commonHintAction("=")
}

func oneOfHintAction() []string {
	return commonHintAction(":")
}

func existenceHintAction() []string {
	return commonHintAction("")
}

func selectHintAction() []string {
	return commonHintAction("")
}

var equalityFilterRegex = regexp.MustCompile(`([^!=<>]+)\s*(=|!=)\s*(.*)`)

func buildEqualityFilters(wheres []string) []common.EqualityFilter {
	filters := make([]common.EqualityFilter, 0, len(wheres))
	for _, whereClause := range wheres {
		//pieces := strings.SplitN(whereClause, "=", 2)
		matches := equalityFilterRegex.FindAllStringSubmatch(whereClause, -1)
		if len(matches) != 1 {
			fmt.Println("Invalid where clause", whereClause)
			os.Exit(1)
		}
		filters = append(filters, common.EqualityFilter{
			FieldName: matches[0][1],
			Operator:  matches[0][2],
			Value:     matches[0][3],
		})
	}
	return filters
}

var membershipFilterRegex = regexp.MustCompile(`([^!=<>]+)\s*:\s*(.*)`)

func buildMembershipFilters(oneOfs []string, notOneOfs []string) []common.MembershipFilter {
	// Build a nested map of field names and their membership constraints
	fields := make(map[string]map[bool][]string)
	for _, oneOfClause := range oneOfs {
		matches := membershipFilterRegex.FindAllStringSubmatch(oneOfClause, -1)
		if len(matches) != 1 {
			fmt.Println("Invalid one-of clause", oneOfClause)
			os.Exit(1)
		}
		fieldName, value := matches[0][1], matches[0][2]
		field, ok := fields[fieldName]
		if !ok {
			fields[fieldName] = map[bool][]string{
				true:  {value},
				false: {},
			}
		} else {
			field[true] = append(fields[fieldName][true], value)
		}
	}
	for _, notOneOfClause := range notOneOfs {
		matches := membershipFilterRegex.FindAllStringSubmatch(notOneOfClause, -1)
		if len(matches) != 1 {
			fmt.Println("Invalid not-one-of clause", notOneOfClause)
			os.Exit(1)
		}
		fieldName, value := matches[0][1], matches[0][2]
		field, ok := fields[fieldName]
		if !ok {
			fields[fieldName] = map[bool][]string{
				false: {value},
				true:  {},
			}
		} else {
			field[false] = append(fields[fieldName][true], value)
		}
	}
	// Create a slice of filters, one per each field name with membership constraints
	filters := make([]common.MembershipFilter, 0, len(fields))
	for fieldName, m := range fields {
		filters = append(filters, common.MembershipFilter{
			FieldName:     fieldName,
			ValidValues:   m[true],
			InvalidValues: m[false],
		})
	}
	return filters
}

func buildExistenceFilters(exists []string, notExists []string) []common.ExistenceFilter {
	filters := make([]common.ExistenceFilter, 0, len(exists)+len(notExists))
	for _, existsFieldName := range exists {
		filters = append(filters, common.ExistenceFilter{
			FieldName: existsFieldName,
			Exists:    true,
		})
	}
	for _, notExistsFieldName := range notExists {
		filters = append(filters, common.ExistenceFilter{
			FieldName: notExistsFieldName,
			Exists:    false,
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
		QueryString:       strings.Join(flags.QueryString, " "),
		Before:            before,
		After:             after,
		EqualityFilters:   buildEqualityFilters(flags.Where),
		ExistenceFilters:  buildExistenceFilters(flags.Exists, flags.NotExists),
		MembershipFilters: buildMembershipFilters(flags.OneOf, flags.NotOneOf),
		SelectFields:      flags.Select,
		Unique:            flags.Unique,
	}
}

func queryMain(ctx context.Context, rc config.RuntimeConfig, client common.Client) {
	query := querySelectorsToQuery(queryFlags)
	query.MaxResults = queryFlagMaxResults
	query.Follow = queryFlagFollow
	seenBeforeHash := make(map[string]bool)
	for message := range complete.GatherCompletionInfo(rc, client.Query(ctx, query)) {
		if query.Unique {
			contentHash := message.ContentHash()
			if seenBeforeHash[contentHash] {
				continue
			}
			seenBeforeHash[contentHash] = true
		}
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
