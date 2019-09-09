package main

import (
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

// checkDone stops the lookback at 5 years
func checkDone(rd int) bool {
	if rd > 140976000 {
		return true
	}
	return false
}

// getLatestGLoballyInner
func getLatestGLoballyInner(numberResults int, backDelta int) ([]opslogEvent, error) {

	scanInput := &dynamodb.ScanInput{
		TableName: aws.String(os.Getenv("DB_TABLE_NAME")),
		ExpressionAttributeNames: map[string]*string{
			"#d": aws.String("DateTime"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":lookback": {
				S: aws.String(strconv.FormatInt(time.Now().Unix()-int64(backDelta), 10)),
			},
		},
		FilterExpression: aws.String("#d >= :lookback"),
	}

	results, err := ddbClient.Scan(scanInput)
	if err != nil {
		return []opslogEvent{}, err
	}

	// TODO: move unmarshal out of inner?
	opslogs := []opslogEvent{}
	err = dynamodbattribute.UnmarshalListOfMaps(results.Items, &opslogs)
	if err != nil {
		return []opslogEvent{}, err
	}

	return opslogs, nil

}
