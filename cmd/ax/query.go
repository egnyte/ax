package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/araddon/dateparse"
	"github.com/egnyte/ax/pkg/backend/common"
	"github.com/egnyte/ax/pkg/complete"
	"github.com/egnyte/ax/pkg/config"
	"github.com/zefhemel/kingpin"
	yaml "gopkg.in/yaml.v2"
)

func addQueryFlags(cmd *kingpin.CmdClause) *common.QuerySelectors {
	flags := &common.QuerySelectors{}
	cmd.Flag("last", "Results from last x minutes, hours, days, months, years. If used after and before are ignored").StringVar(&flags.Last)
	cmd.Flag("before", "Results from before").StringVar(&flags.Before)
	cmd.Flag("after", "Results from after").StringVar(&flags.After)
	cmd.Flag("select", "Fields to select").Short('s').HintAction(selectHintAction).StringsVar(&flags.Select)
	cmd.Flag("where", "Add a filter").Short('w').HintAction(whereHintAction).StringsVar(&flags.Where)
	cmd.Flag("where-one-of", "Add a membership filter (FIELD_NAME:FIELD_VALUE)").HintAction(oneOfHintAction).StringsVar(&flags.OneOf)
	cmd.Flag("where-not-one-of", "Add an inverse membership filter (FIELD_NAME:FIELD_VALUE)").HintAction(oneOfHintAction).StringsVar(&flags.NotOneOf)
	cmd.Flag("where-exists", "Add a field existence filter").HintAction(existenceHintAction).StringsVar(&flags.Exists)
	cmd.Flag("where-not-exists", "Add an inverse field existence filter").HintAction(existenceHintAction).StringsVar(&flags.NotExists)
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

func populateMembershipFilterMap(clauses []string, member bool, fields map[string]map[bool][]string) {
	for _, clause := range clauses {
		matches := membershipFilterRegex.FindAllStringSubmatch(clause, -1)
		if len(matches) != 1 {
			if member {
				fmt.Println("Invalid one-of clause", clause)
			} else {
				fmt.Println("Invalid not-one-of clause", clause)
			}
			os.Exit(1)
		}
		fieldName, value := matches[0][1], matches[0][2]
		field, ok := fields[fieldName]
		if !ok {
			fields[fieldName] = map[bool][]string{
				member:  {value},
				!member: {},
			}
		} else {
			field[member] = append(fields[fieldName][member], value)
		}
	}
}

func buildMembershipFilters(oneOfs []string, notOneOfs []string) []common.MembershipFilter {
	// Build a nested map of field names and their membership constraints
	fields := make(map[string]map[bool][]string)
	populateMembershipFilterMap(oneOfs, true, fields)
	populateMembershipFilterMap(notOneOfs, false, fields)
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

func stringToTime(raw, name string) *time.Time {
	if raw != "" {
		parsed, err := dateparse.ParseLocal(raw)
		if err != nil {
			fmt.Printf("Could not parse %s date: %s\n", name, raw)
			os.Exit(1)
		}
		fmt.Printf("Parsed --%s  as %s", name, parsed.Format(common.TimeFormat))
		return &parsed
	}
	return nil
}

// Do this in order to mock it in test
var timeNow = time.Now

func lastToTimeInterval(raw string) (*time.Time, *time.Time, error) {
	splitted := strings.Split(raw, " ")
	if len(splitted) != 2 {
		return nil, nil, fmt.Errorf("last filter should contain amount and unit")
	}

	amountStr, unit := splitted[0], splitted[1]
	amount, err := strconv.Atoi(amountStr)
	if err != nil {
		return nil, nil, fmt.Errorf("amount has to be numeric")
	}
	before := timeNow()

	var after time.Time
	switch unit {
	case "minutes", "minute", "min":
		after = before.Add(-time.Minute * time.Duration(amount))
	case "hours", "hour", "h":
		after = before.Add(-time.Hour * time.Duration(amount))
	case "days", "day", "d":
		after = before.AddDate(0, 0, -amount)
	case "months", "month", "m":
		after = before.AddDate(0, -amount, 0)
	case "years", "year", "y":
		after = before.AddDate(-amount, 0, 0)
	default:
		return nil, nil, fmt.Errorf("unknown unit")
	}

	return &before, &after, nil
}

func querySelectorsToQuery(flags *common.QuerySelectors) common.Query {
	var before, after *time.Time
	last := flags.Last

	if last != "" {
		before, after, _ = lastToTimeInterval(last)
	} else {
		before = stringToTime(flags.Before, "before")
		after = stringToTime(flags.After, "after")
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
	if !client.ImplementsAdvancedFilters() && (len(query.ExistenceFilters) > 0 || len(query.MembershipFilters) > 0) {
		fmt.Println("This backend does not support advanded filters (yet!)")
		os.Exit(1)
	}

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
		printMessage(message, queryFlagOutputFormat, rc.Config.Colors)
	}

}

func printMessage(message common.LogMessage, queryOutputFormat string, colorConfig config.ColorConfig) {
	switch queryOutputFormat {
	case "text":
		ts := message.Timestamp.Format(common.TimeFormat)
		timestampColor := config.ColorToTermColor(colorConfig.Timestamp)
		fmt.Printf("%s ", timestampColor.Sprintf("[%s]", ts))
		if msg, ok := message.Attributes["message"].(string); ok {
			messageColor := config.ColorToTermColor(colorConfig.Message)
			fmt.Printf("%s ", messageColor.Sprint(msg))
		}
		attributeKeyColor := config.ColorToTermColor(colorConfig.AttributeKey)
		attributeValueColor := config.ColorToTermColor(colorConfig.AttributeValue)
		for key, value := range message.Attributes {
			if key == "message" || value == nil {
				continue
			}
			fmt.Printf("%s%s ", attributeKeyColor.Sprintf("%s=", key), attributeValueColor.Sprintf("%+v", value))
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
