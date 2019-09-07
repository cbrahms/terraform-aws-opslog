package main

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/nlopes/slack"
)

// opslogEvent is the struct for dynamodb
type opslogEvent struct {
	Channel           string
	MessageIdentifier string
	Text              string
	Tags              []string
}

// getUser gets the user string from MessageIdentifier
func (o *opslogEvent) getUser() string {
	re := regexp.MustCompile(`#`)
	identData := re.Split(o.MessageIdentifier, 2)
	return identData[0]
}

func (o *opslogEvent) getDateTime() string {
	re := regexp.MustCompile(`#`)
	identData := re.Split(o.MessageIdentifier, 2)
	i, err := strconv.ParseInt(identData[1], 10, 64)
	if err != nil {
		log.Printf("Error converting unix time to pretty: %s", err.Error())
	}
	return time.Unix(i, 0).String()
}

// fmtTag formats it pretty for fmtChannelAck
func fmtTag(tag string) string {
	re := regexp.MustCompile(`:`)
	tags := re.Split(tag, 2)
	return fmt.Sprintf("*%s:* %s", tags[0], tags[1])
}

// createOpslogEvent converts the slash request to a struct
func createOpslogEvent(req slackRequest) opslogEvent {

	tags := harvestTags(req.text)
	detaggedEvent := detagOrig(req.text, tags)

	return opslogEvent{
		Channel:           req.channelName,
		MessageIdentifier: fmt.Sprintf("%s#%d", req.userName, time.Now().Unix()),
		Text:              detaggedEvent,
		Tags:              tags,
	}
}

// harvestTags infers the dd tags from the original text
func harvestTags(input string) []string {
	var tags []string
	re := regexp.MustCompile(`#\w+:\w+`)
	byteTags := re.FindAll([]byte(input), -1)
	for _, byteTag := range byteTags {
		tag := strings.Replace(string(byteTag), "#", "", -1)
		tags = append(tags, tag)
	}
	return tags
}

// detagOrig removes the dd tags from the original text
func detagOrig(input string, tags []string) string {
	for _, tag := range tags {
		tag = fmt.Sprintf("#%s", tag)
		input = strings.Replace(input, tag, "", -1)
	}
	return strings.TrimSpace(input)
}

// fmtChannelAck formats the ack message with new block messaging from slack
func fmtChannelAck(event opslogEvent) slack.MsgOption {

	var tagBlocks []slack.MixedElement
	tagBlock := slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("*user:* %s", event.getUser()), false, false)
	tagBlocks = append(tagBlocks, tagBlock)
	for _, tag := range event.Tags {
		tagBlock := slack.NewTextBlockObject("mrkdwn", fmtTag(tag), false, false)
		tagBlocks = append(tagBlocks, tagBlock)
	}
	divSection := slack.NewDividerBlock()
	headerText := slack.NewTextBlockObject("mrkdwn", event.Text, false, false)
	headerSection := slack.NewSectionBlock(headerText, nil, nil)
	tagsSection := slack.NewContextBlock(
		"",
		tagBlocks...,
	)
	msg := slack.MsgOptionBlocks(
		divSection,
		headerSection,
		tagsSection,
	)

	return msg
}

// createOpslogEvent converts the raw text to a datadog event and pushes it
// func createOpslogEvent(req slackRequest) dd.Event {

// 	opslogEvent := dd.Event{}
// 	opslogEvent.Tags = []string{
// 		"app:opslog-test",
// 		fmt.Sprintf("channel:%s", req.channelName),
// 		fmt.Sprintf("user:%s", req.userName),
// 	}

// 	tags := harvestTags(req.text)

// 	detaggedEvent := detagOrig(req.text, tags)

// 	opslogEvent.SetTitle(detaggedEvent)
// 	opslogEvent.Tags = append(opslogEvent.Tags, tags...)

// 	return opslogEvent
// }
