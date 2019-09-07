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
	Channel      string
	User         string
	DateTime     string
	Text         string
	AckTimestamp string
	ChannelID    string
	Tags         []string
}

// getDateTime converts the unix timestamp to readable
func (o *opslogEvent) getDateTime() string {
	i, err := strconv.ParseInt(o.DateTime, 10, 64)
	if err != nil {
		log.Printf("Error converting unix timestamp: %s", err.Error())
	}
	return time.Unix(i, 0).String()
}

// createOpslogEvent converts the slash request to a struct
func createOpslogEvent(req slackRequest) opslogEvent {

	tags := harvestTags(req.text)
	detaggedEvent := detagOrig(req.text, tags)

	return opslogEvent{
		Channel:  req.channelName,
		User:     req.userName,
		DateTime: strconv.FormatInt(time.Now().Unix(), 10),
		Text:     detaggedEvent,
		Tags:     tags,
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

// fmtTag formats it pretty for fmtChannelAck
func fmtTag(tag string) string {
	re := regexp.MustCompile(`:`)
	tags := re.Split(tag, 2)
	return fmt.Sprintf("*%s:* %s", tags[0], tags[1])
}

// fmtChannelAck formats the ack message with new block messaging from slack
func fmtChannelAck(event opslogEvent) slack.MsgOption {

	var tagBlocks []slack.MixedElement
	tagBlock := slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("*user:* %s", event.User), false, false)
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
