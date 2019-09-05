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
	userName    string
	text        string
}

// createOpslogEvent converts the raw text to a datadog event and pushes it
func createOpslogEvent(req slackRequest) dd.Event {

	opslogEvent := dd.Event{}
	// TODO: change tag to opslog when done along with tf dd filter
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
func repsond(response string) (events.APIGatewayProxyResponse, error) {
	log.Print(response)
	return events.APIGatewayProxyResponse{
		Body:       response,
		StatusCode: 200,
	}, nil
}

// splitTag formats it pretty for slack
func splitTag(tag string) string {
	re := regexp.MustCompile(`:`)
	tags := re.Split(tag, 2)
	return fmt.Sprintf("*%s:* %s", tags[0], tags[1])
}

// fmtChannelAck formats the ack message with new block messaging from slack
func fmtChannelAck(event dd.Event) slack.MsgOption {

	var tags []slack.MixedElement
	for _, tag := range event.Tags {
		if strings.Contains(tag, "channel:") || strings.Contains(tag, "app:opslog") {
			continue
		}
		fmtdTag := splitTag(tag)
		newTag := slack.NewTextBlockObject("mrkdwn", fmtdTag, false, false)
		tags = append(tags, newTag)
	}
	headerText := slack.NewTextBlockObject("mrkdwn", event.GetTitle(), false, false)
	headerSection := slack.NewSectionBlock(headerText, nil, nil)
	divSection := slack.NewDividerBlock()
	tagsSection := slack.NewContextBlock(
		"",
		tags...,
	)
	msg := slack.MsgOptionBlocks(
		headerSection,
		divSection,
		tagsSection,
	)

	return msg
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	slackAPI := slack.New(os.Getenv("SLACK_TOKEN"))
	ddClient := dd.NewClient(os.Getenv("DD_API_KEY"), os.Getenv("DD_APP_KEY"))
	dashURL := fmt.Sprintf("https://%s.datadoghq.com/dashboard/%s/opslog",
		os.Getenv("DD_TEAM_NAME"), os.Getenv("DD_DASH_ID"))

	vals, _ := url.ParseQuery(request.Body)
	req := slackRequest{
		vals.Get("token"),
		vals.Get("channel_id"),
		vals.Get("channel_name"),
		vals.Get("user_name"),
		vals.Get("text"),
	}

	log.Printf("user %s in chan %s with text: %s", req.userName, req.channelName, req.text)

	token := os.Getenv("SLACK_VERIFICATION_TOKEN")
	if req.token != token {
		return repsond("Invalid token.")
	}

	if len(req.text) > 400 {
		return repsond("Message is over 400 characters, Invalid.")
	}

	if req.channelName == "directmessage" {
		return repsond("No direct messages, Invalid.")
	}

	opslogEvent := createOpslogEvent(req)

	_, err := ddClient.PostEvent(&opslogEvent)
	if err != nil {
		return repsond("Error posting event to datadog")
	}

	_, _, err = slackAPI.PostMessage(req.channelID, fmtChannelAck(opslogEvent))
	if err != nil {
		log.Printf("Slack error: %s", err)
	}

	return events.APIGatewayProxyResponse{
		Body:       fmt.Sprintf("ok, %s", dashURL),
		StatusCode: 200,
	}, nil
}

func main() {
	lambda.Start(handler)
}
