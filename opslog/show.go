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

// getLatestGLobally
func getLatestGLobally(numberResults int) ([]opslogEvent, error) {
	events := []opslogEvent{}
	var err error
	recursDelta := 79200 // ~ 1 day
	for len(events) < numberResults {
		events, err = getLatestGLoballyInner(numberResults, recursDelta)
		if err != nil {
			return []opslogEvent{}, err
		}
		if checkDone(recursDelta) {
			break
		}
		recursDelta = recursDelta * 2
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].DateTime > events[j].DateTime
	})

	if len(events) > numberResults {
		return events[:numberResults], nil
	}

	return events, nil
}

// readableTime
func relatableTime(eventTime string) (string, error) {
	i, err := strconv.ParseInt(eventTime, 10, 64)
	if err != nil {
		return "", err
	}
	tm := time.Unix(i, 0)
	return humanize.Time(tm), nil
}

// evTime
func evTime(eventTime string) (string, error) {
	i, err := strconv.ParseInt(eventTime, 10, 64)
	if err != nil {
		return "", err
	}
	tm := time.Unix(i, 0)
	return tm.Format("Mon Jan 2 3:04:05pm"), nil
}

// fmtUser
func fmtUser(userID string, username string) string {
	if userID != "" {
		return fmt.Sprintf("<@%s>", userID)
	}
	return fmt.Sprintf("@%s", username)
}

// fmtChan
func fmtChan(channelID string, channel string) string {
	if channelID != "" {
		return fmt.Sprintf("<#%s|%s>", channelID, channel)
	}
	return fmt.Sprintf("#%s", channel)
}

func fmtEvents(events []opslogEvent, eventsTitle string) slack.MsgOption {

	var bufferedEvents bytes.Buffer

	for evIndex, ev := range events {

		relatableTime, _ := relatableTime(ev.DateTime)

		eventTime, _ := evTime(ev.DateTime)

		if len(ev.Tags) > 0 {
			fmt.Fprintf(&bufferedEvents, "%d) %s @ %s (%s) in %s w/ %s:\n%s\n",
				evIndex, fmtUser(ev.UserID, ev.User), eventTime, relatableTime, fmtChan(ev.ChannelID, ev.Channel), ev.Tags, ev.Text)
		} else {
			fmt.Fprintf(&bufferedEvents, "%d) %s @ %s (%s) in %s:\n%s\n",
				evIndex, fmtUser(ev.UserID, ev.User), eventTime, relatableTime, fmtChan(ev.ChannelID, ev.Channel), ev.Text)
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
