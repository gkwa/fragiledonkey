package query

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/dustin/go-humanize/english"
	"github.com/gkwa/fragiledonkey/duration"
	"github.com/taylormonacelli/lemondrop"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

type AMI struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	CreationDate time.Time `json:"creation_date"`
	Snapshots    []string  `json:"snapshots"`
	State        string    `json:"state"`
	Region       string    `json:"region"`
}

type SnapshotInfo struct {
	ID          string
	Age         string
	Description string
}

const maxConcurrentRequests = 10

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

func QueryAMIs(client *ec2.Client, pattern, region string) []AMI {
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

func QueryAMIsAllRegions(pattern string) ([]AMI, error) {
	regionDetails, err := lemondrop.GetRegionDetails()
	if err != nil {
		fmt.Println("Error getting region details:", err)
		return nil, err
	}

	ctx := context.Background()
	sem := semaphore.NewWeighted(maxConcurrentRequests)
	var g errgroup.Group
	var mu sync.Mutex
	var allAMIs []AMI

	for _, rd := range regionDetails {
		rd := rd
		err := sem.Acquire(ctx, 1)
		if err != nil {
			continue
		}

		g.Go(func() error {
			defer sem.Release(1)

			cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(rd.Region))
			if err != nil {
				fmt.Printf("Error loading config for region %s: %v\n", rd.Region, err)
				return err
			}

			client := ec2.NewFromConfig(cfg)
			amis := QueryAMIs(client, pattern, rd.Region)

			mu.Lock()
			allAMIs = append(allAMIs, amis...)
			mu.Unlock()

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	fmt.Printf("Found %d %s from %d %s queried\n",
		len(allAMIs),
		english.PluralWord(len(allAMIs), "AMI", ""),
		len(regionDetails),
		english.PluralWord(len(regionDetails), "region", ""))

	return allAMIs, nil
}

func querySnapshotsForAMI(ami AMI, now time.Time) ([]SnapshotInfo, error) {
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(ami.Region))
	if err != nil {
		return nil, fmt.Errorf("error loading config for region %s: %v", ami.Region, err)
	}

	ctx := context.Background()
	sem := semaphore.NewWeighted(maxConcurrentRequests)
	var g errgroup.Group
	var snapshots []SnapshotInfo
	var mu sync.Mutex

	for _, snapshotID := range ami.Snapshots {
		snapshotID := snapshotID
		err := sem.Acquire(ctx, 1)
		if err != nil {
			continue
		}

		g.Go(func() error {
			defer sem.Release(1)

			input := &ec2.DescribeSnapshotsInput{
				SnapshotIds: []string{snapshotID},
			}

			snapshotResult, err := ec2.NewFromConfig(cfg).DescribeSnapshots(ctx, input)
			if err != nil {
				return fmt.Errorf("error describing snapshot %s: %v", snapshotID, err)
			}

			if len(snapshotResult.Snapshots) > 0 {
				snapshot := snapshotResult.Snapshots[0]
				startTime := *snapshot.StartTime
				age := duration.RelativeAge(now.Sub(startTime))

				mu.Lock()
				snapshots = append(snapshots, SnapshotInfo{
					ID:          snapshotID,
					Age:         age,
					Description: *snapshot.Description,
				})
				mu.Unlock()
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return snapshots, nil
}

func RunQueryAllRegions(pattern string) {
	amis, err := QueryAMIsAllRegions(pattern)
	if err != nil {
		fmt.Println("Error querying AMIs across regions:", err)
		return
	}

	now := time.Now()

	ctx := context.Background()
	sem := semaphore.NewWeighted(maxConcurrentRequests)
	var g errgroup.Group

	type amiWithSnapshots struct {
		ami       AMI
		snapshots []SnapshotInfo
	}

	results := make([]amiWithSnapshots, len(amis))

	for i, ami := range amis {
		i, ami := i, ami
		err := sem.Acquire(ctx, 1)
		if err != nil {
			continue
		}

		g.Go(func() error {
			defer sem.Release(1)

			snapshots, err := querySnapshotsForAMI(ami, now)
			if err != nil {
				return err
			}

			results[i] = amiWithSnapshots{ami: ami, snapshots: snapshots}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		fmt.Println("Error querying snapshots:", err)
		return
	}

	for _, result := range results {
		age := duration.RelativeAge(now.Sub(result.ami.CreationDate))
		fmt.Printf("%-5s %-20s %-20s %s\n", age, result.ami.ID, result.ami.Name, result.ami.Region)

		for _, snapshot := range result.snapshots {
			fmt.Printf("    %-5s %-20s %s\n", snapshot.Age, snapshot.ID, snapshot.Description)
		}
	}
}
