package main

import (
	"os"
	"sort"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

// getLatestByUser
func getLatestByUser(user string) (opslogEvent, error) {

	queryInput := &dynamodb.QueryInput{
		TableName: aws.String(os.Getenv("DB_TABLE_NAME")),
		IndexName: aws.String("UserIndex"),
		KeyConditions: map[string]*dynamodb.Condition{
			"User": {
				ComparisonOperator: aws.String("EQ"),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(user),
					},
				},
			},
		},
	}

	results, err := ddbClient.Query(queryInput)

	if err != nil {
		return opslogEvent{}, err
	}

	opslogs := []opslogEvent{}
	err = dynamodbattribute.UnmarshalListOfMaps(results.Items, &opslogs)
	if err != nil {
		return opslogEvent{}, err
	}

	sort.Slice(opslogs, func(i, j int) bool {
		return opslogs[i].DateTime > opslogs[j].DateTime
	})

	return opslogs[0], nil
}

func deleteOpslog(doomedOpslog opslogEvent) error {

	input := &dynamodb.DeleteItemInput{
		TableName: aws.String(os.Getenv("DB_TABLE_NAME")),
		Key: map[string]*dynamodb.AttributeValue{
			"Channel": {
				S: aws.String(doomedOpslog.Channel),
			},
			"DateTime": {
				S: aws.String(doomedOpslog.DateTime),
			},
		},
	}

	_, err := ddbClient.DeleteItem(input)
	if err != nil {
		return err
	}

	return nil
}
