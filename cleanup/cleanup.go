package cleanup

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/spf13/viper"
	"github.com/taylormonacelli/fragiledonkey/duration"
	"github.com/taylormonacelli/fragiledonkey/query"
	"github.com/taylormonacelli/lemondrop"
	"golang.org/x/sync/errgroup"
)

func RunCleanup(olderThan, newerThan string, assumeYes bool, pattern string) {
	var olderThanDuration time.Duration
	var newerThanDuration time.Duration
	var err error

	if olderThan != "" {
		olderThanDuration, err = duration.ParseDuration(olderThan)
		if err != nil {
			fmt.Println("Error parsing older-than duration:", err)
			return
		}
	}

	if newerThan != "" {
		newerThanDuration, err = duration.ParseDuration(newerThan)
		if err != nil {
			fmt.Println("Error parsing newer-than duration:", err)
			return
		}
	}

	regionDetails, err := lemondrop.GetRegionDetails()
	if err != nil {
		fmt.Println("Error getting region details:", err)
		return
	}

	var g errgroup.Group
	for _, rd := range regionDetails {
		rd := rd
		g.Go(func() error {
			cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(rd.Region))
			if err != nil {
				fmt.Printf("Error loading config for region %s: %v\n", rd.Region, err)
				return err
			}

			client := ec2.NewFromConfig(cfg)
			cleanupRegion(client, olderThanDuration, newerThanDuration, assumeYes, pattern, rd.Region)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		fmt.Println("Error during cleanup:", err)
	}
}

func cleanupRegion(client *ec2.Client, olderThanDuration, newerThanDuration time.Duration, assumeYes bool, pattern string, region string) {
	amis := query.QueryAMIs(client, pattern, region)

	now := time.Now()
	var imagesToDelete []string
	var snapshotsToDelete []string

	for _, ami := range amis {
		if ami.State != "available" {
			continue
		}

		if olderThanDuration != 0 && now.Sub(ami.CreationDate) > olderThanDuration {
			imagesToDelete = append(imagesToDelete, ami.ID)
			snapshotsToDelete = append(snapshotsToDelete, ami.Snapshots...)
		} else if newerThanDuration != 0 && now.Sub(ami.CreationDate) < newerThanDuration {
			imagesToDelete = append(imagesToDelete, ami.ID)
			snapshotsToDelete = append(snapshotsToDelete, ami.Snapshots...)
		}
	}

	if len(imagesToDelete) == 0 && len(snapshotsToDelete) == 0 {
		if viper.GetBool("verbose") {
			fmt.Printf("No AMIs or snapshots to delete in region %s.\n", region)
		}
		return
	}

	fmt.Printf("AMIs to be deleted in region %s:\n", region)
	for _, imageID := range imagesToDelete {
		fmt.Println("-", imageID)
	}

	fmt.Printf("Snapshots to be deleted in region %s:\n", region)
	for _, snapshotID := range snapshotsToDelete {
		fmt.Println("-", snapshotID)
	}

	if !assumeYes {
		fmt.Print("Do you want to proceed with the deletion? (y/n): ")
		var confirm string
		fmt.Scanln(&confirm)

		if confirm != "y" {
			fmt.Println("Aborting deletion.")
			return
		}
	}

	for _, imageID := range imagesToDelete {
		input := &ec2.DeregisterImageInput{
			ImageId: aws.String(imageID),
		}

		_, err := client.DeregisterImage(context.Background(), input)
		if err != nil {
			fmt.Printf("Error deregistering AMI %s: %v\n", imageID, err)
			continue
		}

		fmt.Printf("Deregistered AMI: %s\n", imageID)
	}

	for _, snapshotID := range snapshotsToDelete {
		input := &ec2.DeleteSnapshotInput{
			SnapshotId: aws.String(snapshotID),
		}

		_, err := client.DeleteSnapshot(context.Background(), input)
		if err != nil {
			fmt.Printf("Error deleting snapshot %s: %v\n", snapshotID, err)
			continue
		}

		fmt.Printf("Deleted snapshot: %s\n", snapshotID)
	}

	fmt.Printf("Cleanup completed in region %s.\n", region)
}
