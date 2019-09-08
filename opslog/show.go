package main

import (
	"bytes"
	"fmt"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/dustin/go-humanize"
	"github.com/nlopes/slack"
)

// getLatestByChannel
func getLatestByChannel(channel string, numberResults int) ([]opslogEvent, error) {

	queryInput := &dynamodb.QueryInput{
		TableName: aws.String(os.Getenv("DB_TABLE_NAME")),
		KeyConditions: map[string]*dynamodb.Condition{
			"Channel": {
				ComparisonOperator: aws.String("EQ"),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(channel),
					},
				},
			},
		},
	}

	results, err := ddbClient.Query(queryInput)

	if err != nil {
		return []opslogEvent{}, err
	}

	opslogs := []opslogEvent{}
	err = dynamodbattribute.UnmarshalListOfMaps(results.Items, &opslogs)
	if err != nil {
		return []opslogEvent{}, err
	}

	sort.Slice(opslogs, func(i, j int) bool {
		return opslogs[i].DateTime > opslogs[j].DateTime
	})

	if len(opslogs) > numberResults {
		return opslogs[:numberResults], nil
	}

	return opslogs, nil
}

// readableTime
func readableTime(eventTime string) (string, error) {
	i, err := strconv.ParseInt(eventTime, 10, 64)
	if err != nil {
		return "", err
	}
	tm := time.Unix(i, 0)
	return humanize.Time(tm), nil
}

func fmtEvents(events []opslogEvent, eventsTitle string) slack.MsgOption {

	var bufferedEvents bytes.Buffer

	for _, ev := range events {
		legibleDateTime, _ := readableTime(ev.DateTime)
		if len(ev.Tags) > 0 {
			fmt.Fprintf(&bufferedEvents, "%s <@%s> in <#%s|%s>: `%s` with tags `%s`\n",
				legibleDateTime, ev.UserID, ev.ChannelID, ev.Channel, ev.Text, ev.Tags)
		} else {
			fmt.Fprintf(&bufferedEvents, "%s <@%s> in <#%s|%s>: `%s`\n",
				legibleDateTime, ev.UserID, ev.ChannelID, ev.Channel, ev.Text)
		}
	}

	divSection := slack.NewDividerBlock()
	headerText := slack.NewTextBlockObject("mrkdwn", eventsTitle, false, false)
	headerSection := slack.NewSectionBlock(headerText, nil, nil)
	eventsText := slack.NewTextBlockObject("mrkdwn", bufferedEvents.String(), false, false)
	eventsSection := slack.NewSectionBlock(eventsText, nil, nil)

	msg := slack.MsgOptionBlocks(
		divSection,
		headerSection,
		divSection,
		eventsSection,
	)

	return msg
}
