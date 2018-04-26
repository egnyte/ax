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
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
)

func addCompletionFunc(flagSet *pflag.FlagSet, flagName, bashCompletionFunction string) {
	flagSet.Lookup(flagName).Annotations = map[string][]string{cobra.BashCompCustom: {bashCompletionFunction}}
}

func addQueryFlags(cmd *cobra.Command) *common.QuerySelectors {
	flags := &common.QuerySelectors{}
	cFlags := cmd.Flags()
	cFlags.StringVar(&flags.Before, "before", "", "Results from before")
	cFlags.StringVar(&flags.After, "after", "", "Results from after")
	cFlags.StringArrayVarP(&flags.Select, "select", "s", []string{}, "Select specific attributes only")
	addCompletionFunc(cFlags, "select", "__ax_get_attrs_select")
	cFlags.StringArrayVarP(&flags.Where, "where", "w", []string{}, "Add a filter")
	addCompletionFunc(cFlags, "where", "__ax_get_attrs_where")
	cFlags.StringArrayVar(&flags.OneOf, "where-one-of", []string{}, "Add a membership filter (FIELD_NAME:FIELD_VALUE)")
	cFlags.StringArrayVar(&flags.NotOneOf, "where-not-one-of", []string{}, "Add a negative membership filter (FIELD_NAME:FIELD_VALUE)")
	addCompletionFunc(cFlags, "where-one-of", "__ax_get_attrs_where2")
	addCompletionFunc(cFlags, "where-not-one-of", "__ax_get_attrs_where2")
	cFlags.StringArrayVar(&flags.Exists, "where-exists", []string{}, "Add a field existence filter")
	addCompletionFunc(cFlags, "where-exists", "__ax_get_attrs_select")
	cFlags.StringArrayVar(&flags.NotExists, "where-not-exists", []string{}, "Add a negative field existence filter")
	addCompletionFunc(cFlags, "where-not-exists", "__ax_get_attrs_select")
	cFlags.BoolVar(&flags.Unique, "uniq", false, "Unique log messages only")
	return flags
}

var (
	queryFlags            = addQueryFlags(rootCmd)
	queryFlagMaxResults   int
	queryFlagOutputFormat string
	queryFlagFollow       bool
)

func init() {
	// Hack in the main run command
	rootCmd.Run = func(cmd *cobra.Command, args []string) {
		rc := config.BuildConfig(defaultEnvFlag, dockerFlag)
		client := determineClient(rc.Env)
		if bashScriptFlag {
			fmt.Println(`# To load: eval "$(ax bash-completion)"`)
			rootCmd.GenBashCompletion(os.Stdout)
			return
		}
		ctx := sigtermContextHandler(context.Background())
		if client == nil {
			if len(rc.Config.Environments) == 0 {
				// Assuming first time use
				fmt.Println("Welcome to ax! It looks like this is the first time running, so let's start with creating a new environment.")
				config.AddEnv()
				return
			}
			fmt.Println("No default environment set, please use the --env flag to set one. Exiting.")
			return
		}
		queryMain(ctx, rc, client, args)
	}
	flags := rootCmd.Flags()
	flags.BoolVarP(&queryFlagFollow, "follow", "f", false, "Follow logs in quasi real-time, similar to tail -f")
	flags.IntVarP(&queryFlagMaxResults, "results", "n", 50, "Maximum number of results")
	flags.StringVarP(&queryFlagOutputFormat, "output", "o", "text", "Output format: text|json|yaml")
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

func queryMain(ctx context.Context, rc config.RuntimeConfig, client common.Client, queryPhrase []string) {
	query := querySelectorsToQuery(queryFlags)
	if !client.ImplementsAdvancedFilters() && (len(query.ExistenceFilters) > 0 || len(query.MembershipFilters) > 0) {
		fmt.Println("This backend does not support advanded filters (yet!)")
		os.Exit(1)
	}

	query.MaxResults = queryFlagMaxResults
	query.Follow = queryFlagFollow
	query.QueryString = strings.Join(queryPhrase, " ")
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
