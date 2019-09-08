package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/nlopes/slack"
)

// slackRequest are the important fields we care about from the full slash cmd
type slackRequest struct {
	token       string
	channelID   string
	channelName string
	userID      string
	userName    string
	text        string
}

var ddbClient *dynamodb.DynamoDB
var slackClient *slack.Client

// init clients
func init() {
	region := os.Getenv("AWS_REGION")
	if session, err := session.NewSession(&aws.Config{
		Region: &region,
	}); err != nil {
		log.Printf("Failed to connect to AWS: %s", err.Error())
	} else {
		ddbClient = dynamodb.New(session)
	}
	slackClient = slack.New(os.Getenv("SLACK_OAUTH_TOKEN"))
}

// AWS lambda safe response wrapper + logging return body & ephimeral reason reply
func respond(response string) (events.APIGatewayProxyResponse, error) {
	log.Print(response)
	return events.APIGatewayProxyResponse{
		Body:       response,
		StatusCode: 200,
	}, nil
}

// AWS lambda safe response wrapper 200 only
func ok() (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
	}, nil
}

// getResultCount pulls the results count number from the command
func getResultCount(input string) (int, error) {

	splitCmd := strings.Split(input, " ")

	if len(splitCmd) == 2 {
		r, err := strconv.Atoi(string(splitCmd[1]))
		if err != nil {
			return 0, err
		}
		return r, nil
	}
	return 10, nil
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

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

	reShow := regexp.MustCompile(`^show \d+$`)
	reShowDefault := regexp.MustCompile(`^show$`)
	reShowAll := regexp.MustCompile(`^showall \d+$`)
	reShowAllDefault := regexp.MustCompile(`^showall$`)
	reSearch := regexp.MustCompile(`^search .*$`)
	reSearchAll := regexp.MustCompile(`^searchall .*$`)

	switch {

	// help
	case req.text == "help":

		msg := fmtSendHelp()
		_, err := slackClient.PostEphemeral(req.channelID, req.userID, msg)
		if err != nil {
			return respond(fmt.Sprintf("Slack error: %s", err.Error()))
		}
		return ok()

	// deletelast
	case req.text == "deletelast":

		lastOpslog, err := getLatestByUser(req.userName)
		if err != nil {
			return respond(fmt.Sprintf("Get latest error: %s", err.Error()))
		}
		err = deleteOpslog(lastOpslog)
		if err != nil {
			return respond(fmt.Sprintf("delete item error: %s", err.Error()))
		}
		_, _, err = slackClient.DeleteMessage(lastOpslog.ChannelID, lastOpslog.AckTimestamp)
		if err != nil {
			return respond(fmt.Sprintf("Slack error: %s", err.Error()))
		}
		return respond(fmt.Sprintf("deleted your last opslog entry in <#%s|%s>", lastOpslog.ChannelID, lastOpslog.Channel))

	// show x
	case reShow.Match([]byte(req.text)) || reShowDefault.Match([]byte(req.text)):

		rc, err := getResultCount(req.text)
		if err != nil {
			return respond(fmt.Sprintf("Parse string to int error: %s", err.Error()))
		}

		events, err := getLatestByChannel(req.channelName, rc)
		if err != nil {
			return respond(fmt.Sprintf("Get latest error: %s", err.Error()))
		}

		msg := fmtEvents(events, fmt.Sprintf("*latest %d events in #%s*", len(events), req.channelName))
		_, err = slackClient.PostEphemeral(req.channelID, req.userID, msg)
		if err != nil {
			return respond(fmt.Sprintf("Slack error: %s", err.Error()))
		}
		return ok()

	// showall x
	case reShowAll.Match([]byte(req.text)) || reShowAllDefault.Match([]byte(req.text)):

		return respond("show last x across all channels")

	// search
	case reSearch.Match([]byte(req.text)):

		return respond("search current channel")

	// searchall
	case reSearchAll.Match([]byte(req.text)):

		return respond("search across all channels")

	// new opslog entry
	default:
		if req.channelName == "directmessage" {
			return respond("No direct messages, Invalid.")
		}

		opslogEvent := createOpslogEvent(req)

		_, ackTimestamp, err := slackClient.PostMessage(req.channelID, fmtChannelAck(opslogEvent))
		if err != nil {
			return respond(fmt.Sprintf("Slack error: %s", err.Error()))
		}

		opslogEvent.AckTimestamp = ackTimestamp
		opslogEvent.ChannelID = req.channelID

		marshOpslogEvent, err := dynamodbattribute.MarshalMap(opslogEvent)
		if err != nil {
			return respond(fmt.Sprintf("Error marshalling new opslog: %s", err.Error()))
		}
		input := &dynamodb.PutItemInput{
			Item:      marshOpslogEvent,
			TableName: aws.String(os.Getenv("DB_TABLE_NAME")),
		}

		_, err = ddbClient.PutItem(input)
		if err != nil {
			return respond(fmt.Sprintf("Error putting new opslog in dynamodb: %s", err.Error()))
		}

		return ok()
	}
}

func main() {
	lambda.Start(handler)
}
