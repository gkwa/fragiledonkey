package query

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/dustin/go-humanize/english"
	"github.com/gkwa/fragiledonkey/duration"
	"github.com/spf13/viper"
	"github.com/taylormonacelli/lemondrop"
	"golang.org/x/sync/errgroup"
)

type AMI struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	CreationDate time.Time `json:"creation_date"`
	Snapshots    []string  `json:"snapshots"`
	State        string    `json:"state"`
	Region       string    `json:"region"`
}

var ignoreStatusCodes = []int{
	401, // don't show me errors when I don't have access to region
}

func isIgnoredError(err error) bool {
	for _, code := range ignoreStatusCodes {
		if strings.Contains(err.Error(), fmt.Sprintf("StatusCode: %d", code)) {
			return true
		}
	}

	return false
}

func QueryAMIs(client *ec2.Client, pattern string, region string) []AMI {
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
		if !isIgnoredError(err) {
			fmt.Printf("Error describing images in region %s: %v\n", region, err)
		}

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
			Region:       region,
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
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(viper.GetString("region")))
	if err != nil {
		fmt.Println("Error loading config:", err)
		return
	}

	client := ec2.NewFromConfig(cfg)

	amis := QueryAMIs(client, pattern, viper.GetString("region"))

	now := time.Now()
	for _, ami := range amis {
		age := duration.RelativeAge(now.Sub(ami.CreationDate))
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
				age := duration.RelativeAge(now.Sub(startTime))
				fmt.Printf("    %-5s %-20s %s\n", age, *snapshot.SnapshotId, *snapshot.Description)
			}
		}
	}
}

func QueryAMIsAllRegions(pattern string) ([]AMI, error) {
	regionDetails, err := lemondrop.GetRegionDetails()
	if err != nil {
		fmt.Println("Error getting region details:", err)
		return nil, err
	}

	var g errgroup.Group

	amiChan := make(chan []AMI)

	for _, rd := range regionDetails {
		rd := rd

		g.Go(func() error {
			cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(rd.Region))
			if err != nil {
				fmt.Printf("Error loading config for region %s: %v\n", rd.Region, err)
				return err
			}

			client := ec2.NewFromConfig(cfg)
			amis := QueryAMIs(client, pattern, rd.Region)
			amiChan <- amis

			return nil
		})
	}

	var allAMIs []AMI

	go func() {
		for amis := range amiChan {
			allAMIs = append(allAMIs, amis...)
		}
	}()

	if err := g.Wait(); err != nil {
		return nil, err
	}

	close(amiChan)

	fmt.Printf("Found %d %s from %d %s queried\n", len(allAMIs), english.PluralWord(len(allAMIs), "AMI", ""), len(regionDetails), english.PluralWord(len(regionDetails), "region", ""))

	return allAMIs, nil
}

func RunQueryAllRegions(pattern string) {
	amis, err := QueryAMIsAllRegions(pattern)
	if err != nil {
		fmt.Println("Error querying AMIs across regions:", err)
		return
	}

	now := time.Now()
	for _, ami := range amis {
		age := duration.RelativeAge(now.Sub(ami.CreationDate))
		fmt.Printf("%-5s %-20s %-20s %s\n", age, ami.ID, ami.Name, ami.Region)

		for _, snapshotID := range ami.Snapshots {
			cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(ami.Region))
			if err != nil {
				fmt.Printf("Error loading config for region %s: %v\n", ami.Region, err)
				continue
			}

			input := &ec2.DescribeSnapshotsInput{
				SnapshotIds: []string{snapshotID},
			}

			snapshotResult, err := ec2.NewFromConfig(cfg).DescribeSnapshots(context.Background(), input)
			if err != nil {
				fmt.Println("Error describing snapshot:", err)
				continue
			}

			if len(snapshotResult.Snapshots) > 0 {
				snapshot := snapshotResult.Snapshots[0]
				startTime := *snapshot.StartTime
				age := duration.RelativeAge(now.Sub(startTime))
				fmt.Printf("    %-5s %-20s %s\n", age, *snapshot.SnapshotId, *snapshot.Description)
			}
		}
	}
}
