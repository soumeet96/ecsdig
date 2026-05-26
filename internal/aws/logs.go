package aws

import (
	"context"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
)

type LogLine struct {
	Timestamp time.Time
	Message   string
}

func GetLastLogLines(ctx context.Context, client *cloudwatchlogs.Client, logGroup, streamPrefix, containerName string, n int) ([]LogLine, error) {
	// find the most recent log stream matching the prefix
	prefix := streamPrefix + "/" + containerName
	streamsOut, err := client.DescribeLogStreams(ctx, &cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName:        aws.String(logGroup),
		LogStreamNamePrefix: aws.String(prefix),
		OrderBy:             "LastEventTime",
		Descending:          aws.Bool(true),
		Limit:               aws.Int32(1),
	})
	if err != nil {
		return nil, err
	}
	if len(streamsOut.LogStreams) == 0 {
		return nil, nil
	}

	streamName := aws.ToString(streamsOut.LogStreams[0].LogStreamName)

	eventsOut, err := client.GetLogEvents(ctx, &cloudwatchlogs.GetLogEventsInput{
		LogGroupName:  aws.String(logGroup),
		LogStreamName: aws.String(streamName),
		StartFromHead: aws.Bool(false),
		Limit:         aws.Int32(int32(n)),
	})
	if err != nil {
		return nil, err
	}

	var lines []LogLine
	for _, e := range eventsOut.Events {
		lines = append(lines, LogLine{
			Timestamp: time.UnixMilli(aws.ToInt64(e.Timestamp)),
			Message:   aws.ToString(e.Message),
		})
	}

	sort.Slice(lines, func(i, j int) bool {
		return lines[i].Timestamp.Before(lines[j].Timestamp)
	})

	return lines, nil
}
