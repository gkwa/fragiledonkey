package query

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/spf13/viper"
)

type AMI struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	CreationDate time.Time `json:"creation_date"`
	Snapshots    []string  `json:"snapshots"`
	State        string    `json:"state"`
}

func QueryAMIs(client *ec2.Client, pattern string) []AMI {
	input := &ec2.DescribeImagesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("name"),
				Values: []string{pattern},
			},
			{
				Name:   aws.String("state"),
				Values: []string{"available"},
			},
		},
		Owners: []string{"self"},
	}

	result, err := client.DescribeImages(context.Background(), input)
	if err != nil {
		fmt.Println("Error describing images:", err)
		return nil
	}

	var amis []AMI
	for _, image := range result.Images {
		creationTime, err := time.Parse(time.RFC3339, *image.CreationDate)
		if err != nil {
			fmt.Println("Error parsing creation date:", err)
			continue
		}

		ami := AMI{
			ID:           *image.ImageId,
			Name:         *image.Name,
			CreationDate: creationTime,
			State:        string(image.State),
		}

		input := &ec2.DescribeSnapshotsInput{
			Filters: []types.Filter{
				{
					Name:   aws.String("description"),
					Values: []string{fmt.Sprintf("*%s*", *image.ImageId)},
				},
				{
					Name:   aws.String("status"),
					Values: []string{"completed"},
				},
			},
			OwnerIds: []string{"self"},
		}

		snapshotResult, err := client.DescribeSnapshots(context.Background(), input)
		if err != nil {
			fmt.Println("Error describing snapshots:", err)
			continue
		}

		for _, snapshot := range snapshotResult.Snapshots {
			ami.Snapshots = append(ami.Snapshots, *snapshot.SnapshotId)
		}

		amis = append(amis, ami)
	}

	sort.Slice(amis, func(i, j int) bool {
		return amis[i].CreationDate.After(amis[j].CreationDate)
	})

	return amis
}

func RunQuery(pattern string) {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		fmt.Println("Error loading config:", err)
		return
	}

	region := viper.GetString("region")
	client := ec2.NewFromConfig(cfg, func(o *ec2.Options) {
		o.Region = region
	})

	amis := QueryAMIs(client, pattern)

	now := time.Now()
	for _, ami := range amis {
		age := relativeAge(now.Sub(ami.CreationDate))
		fmt.Printf("%-5s %-20s %s\n", age, ami.ID, ami.Name)

		for _, snapshotID := range ami.Snapshots {
			input := &ec2.DescribeSnapshotsInput{
				SnapshotIds: []string{snapshotID},
			}

			snapshotResult, err := client.DescribeSnapshots(context.Background(), input)
			if err != nil {
				fmt.Println("Error describing snapshot:", err)
				continue
			}

			if len(snapshotResult.Snapshots) > 0 {
				snapshot := snapshotResult.Snapshots[0]
				startTime := *snapshot.StartTime
				age := relativeAge(now.Sub(startTime))
				fmt.Printf("    %-5s %-20s %s\n", age, *snapshot.SnapshotId, *snapshot.Description)
			}
		}
	}
}

func relativeAge(duration time.Duration) string {
	if duration < time.Minute {
		return fmt.Sprintf("%ds", int(duration.Seconds()))
	} else if duration < time.Hour {
		return fmt.Sprintf("%dm", int(duration.Minutes()))
	} else if duration < 24*time.Hour {
		return fmt.Sprintf("%dh", int(duration.Hours()))
	} else if duration < 30*24*time.Hour {
		return fmt.Sprintf("%dd", int(duration.Hours()/24))
	} else if duration < 365*24*time.Hour {
		return fmt.Sprintf("%dM", int(duration.Hours()/(30*24)))
	} else {
		return fmt.Sprintf("%dy", int(duration.Hours()/(365*24)))
	}
}
