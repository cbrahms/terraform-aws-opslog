package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/nlopes/slack"
	dd "gopkg.in/zorkian/go-datadog-api.v2"
)

// slackRequest are the important fields we care about from the full slack request
type slackRequest struct {
	token       string
	channelID   string
	channelName string
	userID      string
	userName    string
	text        string
}

// createOpslogEvent converts the raw text to a datadog event and pushes it
func createOpslogEvent(req slackRequest) dd.Event {

	opslogEvent := dd.Event{}
	opslogEvent.Tags = []string{
		"app:opslog-test",
		fmt.Sprintf("channel:%s", req.channelName),
		fmt.Sprintf("user:%s", req.userName),
	}

	tags := harvestTags(req.text)

	detaggedEvent := detagOrig(req.text, tags)

	opslogEvent.SetTitle(detaggedEvent)
	opslogEvent.Tags = append(opslogEvent.Tags, tags...)

	return opslogEvent
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

// AWS lambda safe response wrapper + logging return body
func respond(response string) (events.APIGatewayProxyResponse, error) {
	log.Print(response)
	return events.APIGatewayProxyResponse{
		Body:       response,
		StatusCode: 200,
	}, nil
}

// fmtTag formats it pretty for slack
func fmtTag(tag string) string {
	re := regexp.MustCompile(`:`)
	tags := re.Split(tag, 2)
	return fmt.Sprintf("*%s:* %s", tags[0], tags[1])
}

// fmtChannelAck formats the ack message with new block messaging from slack
func fmtChannelAck(event dd.Event) slack.MsgOption {

	var tagBlocks []slack.MixedElement
	for _, tag := range event.Tags {
		if strings.Contains(tag, "channel:") || strings.Contains(tag, "app:opslog") {
			continue
		}
		tagBlock := slack.NewTextBlockObject("mrkdwn", fmtTag(tag), false, false)
		tagBlocks = append(tagBlocks, tagBlock)
	}
	divSection := slack.NewDividerBlock()
	headerText := slack.NewTextBlockObject("mrkdwn", event.GetTitle(), false, false)
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

// sendHelp does what it sounds like
func fmtSendHelp(req slackRequest, dashURL string, slackAPI *slack.Client) {

	divSection := slack.NewDividerBlock()
	headerText := slack.NewTextBlockObject("mrkdwn", "*help*", false, false)
	headerSection := slack.NewSectionBlock(headerText, nil, nil)
	divSection = slack.NewDividerBlock()
	opslogText := slack.NewTextBlockObject(
		"mrkdwn",
		"*/opslog <entry> [#tag:value]*\n\tCreate a new opslog entry, optionally adding tags",
		false,
		false,
	)
	opslogSection := slack.NewSectionBlock(opslogText, nil, nil)
	deleteText := slack.NewTextBlockObject(
		"mrkdwn",
		"*/opslog deletelast*\n\tDelete the previous opslog entry created by you",
		false,
		false,
	)
	deleteSection := slack.NewSectionBlock(deleteText, nil, nil)
	showText := slack.NewTextBlockObject(
		"mrkdwn",
		"*/opslog show [x]*\n\tList the previous x opslog entries in the channel it's called from, defaults to 10",
		false,
		false,
	)
	showSection := slack.NewSectionBlock(showText, nil, nil)
	showAllText := slack.NewTextBlockObject(
		"mrkdwn",
		"*/opslog showall [x]*\n\tList the previous x opslog entries globally, defaults to 10",
		false,
		false,
	)
	showAllSection := slack.NewSectionBlock(showAllText, nil, nil)
	searchText := slack.NewTextBlockObject(
		"mrkdwn",
		"*/opslog search <entry>*\n\tSearch for opslog entries in the channel it's called from, limited to 50 results",
		false,
		false,
	)
	searchSection := slack.NewSectionBlock(searchText, nil, nil)
	searchAllText := slack.NewTextBlockObject(
		"mrkdwn",
		"*/opslog searchall <entry>*\n\tSearch for opslog entries globally, limited to 50 results",
		false,
		false,
	)
	searchAllSection := slack.NewSectionBlock(searchAllText, nil, nil)

	msg := slack.MsgOptionBlocks(
		divSection,
		headerSection,
		divSection,
		opslogSection,
		deleteSection,
		showSection,
		showAllSection,
		searchSection,
		searchAllSection,
	)
	_, err := slackAPI.PostEphemeral(req.channelID, req.userID, msg)
	if err != nil {
		log.Printf("Slack error: %s", err)
	}
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	slackAPI := slack.New(os.Getenv("SLACK_OAUTH_TOKEN"))
	ddClient := dd.NewClient(os.Getenv("DD_API_KEY"), os.Getenv("DD_APP_KEY"))
	dashURL := fmt.Sprintf("https://%s.datadoghq.com/dashboard/%s/opslog",
		os.Getenv("DD_TEAM_NAME"), os.Getenv("DD_DASH_ID"))

	vals, _ := url.ParseQuery(request.Body)
	req := slackRequest{
		vals.Get("token"),
		vals.Get("channel_id"),
		vals.Get("channel_name"),
		vals.Get("user_id"),
		vals.Get("user_name"),
		vals.Get("text"),
	}

	log.Printf("user %s in chan %s with text: %s", req.userName, req.channelName, req.text)

	token := os.Getenv("SLACK_VERIFICATION_TOKEN")
	if req.token != token {
		return respond("Invalid token.")
	}

	if len(req.text) > 400 {
		return respond("Message is over 400 characters, Invalid.")
	}

	reSx := regexp.MustCompile(`^show \d+$`)
	reDL := regexp.MustCompile(`^deletelast$`)
	switch {
	case req.text == "help":
		fmtSendHelp(req, dashURL, slackAPI)
		return events.APIGatewayProxyResponse{
			StatusCode: 200,
		}, nil
	case reSx.Match([]byte(req.text)):
		return respond("show")
	case reDL.Match([]byte(req.text)):
		return respond("delete last")
	default:
		if req.channelName == "directmessage" {
			return respond("No direct messages, Invalid.")
		}

		opslogEvent := createOpslogEvent(req)

		_, err := ddClient.PostEvent(&opslogEvent)
		if err != nil {
			return respond("Error posting event to datadog")
		}

		_, _, err = slackAPI.PostMessage(req.channelID, fmtChannelAck(opslogEvent))
		if err != nil {
			log.Printf("Slack error: %s", err)
		}

		return events.APIGatewayProxyResponse{
			StatusCode: 200,
		}, nil
	}
}

func main() {
	lambda.Start(handler)
}
