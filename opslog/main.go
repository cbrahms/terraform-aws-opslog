package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"regexp"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
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

var ddbClient *dynamodb.DynamoDB
var slackClient *slack.Client
var ddClient *dd.Client
var dashURL string

// init all the clients, prime dash URL
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
	ddClient = dd.NewClient(os.Getenv("DD_API_KEY"), os.Getenv("DD_APP_KEY"))
	dashURL = fmt.Sprintf("https://%s.datadoghq.com/dashboard/%s/opslog",
		os.Getenv("DD_TEAM_NAME"), os.Getenv("DD_DASH_ID"))
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
	reShowAll := regexp.MustCompile(`^showall \d+$`)
	reSearch := regexp.MustCompile(`^search .*$`)
	reSearchAll := regexp.MustCompile(`^searchall .*$`)

	switch {

	// help
	case req.text == "help":

		msg := fmtSendHelp(req, dashURL)
		_, err := slackClient.PostEphemeral(req.channelID, req.userID, msg)
		if err != nil {
			log.Printf("Slack error: %s", err.Error())
		}
		return ok()

	// deletelast
	case req.text == "deletelast":

		lastEntry := getLastOpslog(req.userName)
		return respond(lastEntry)

	// show x
	case reShow.Match([]byte(req.text)):

		return respond("show last x in current channel")

	// showall x
	case reShowAll.Match([]byte(req.text)):

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

		_, _, err = slackClient.PostMessage(req.channelID, fmtChannelAck(opslogEvent))
		if err != nil {
			return respond(fmt.Sprintf("Slack error: %s", err.Error()))
		}

		return ok()
	}
}

func main() {
	lambda.Start(handler)
}
