package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/endpoints"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

const (
	ubuntuImageSearch = "ubuntu/images/hvm-ssd/ubuntu-xenial-16.04-amd64*"
	keyName           = "aws-sdk-gov2-key"
	ec2Type           = "t2.medium"
)

func main() {

	// get ec2 connection
	ec2Svc, err := getEC2Client(endpoints.UsEast1RegionID)
	if err != nil {
		exitErrorf("Unable to get EC2 Client, %v", err)
	}
	fmt.Printf("AWS connection %#v\n", ec2Svc)

	// get latest ubuntu image
	ubuntuAMI, err := findUbuntuAMI(ec2Svc)
	if err != nil {
		exitErrorf(
			"Unable to find latest ubuntu image using search query [%s], %v",
			ubuntuImageSearch, err)
	}
	fmt.Printf("AMI %s\n", ubuntuAMI)

	// create key pair
	err = createSSHKeyPair(ec2Svc)
	if err != nil {
		exitErrorf(
			"Unable to create SSH Key Pair, %v", err)
	}

	// run instance
	instanceID, err := runInstance(ec2Svc, ubuntuAMI)
	if err != nil {
		exitErrorf(
			"Unable to run instance, %v", err)
	}

	// get instance public IP
	pubIP, err := getInstancePublicIP(ec2Svc, instanceID)
	if err != nil {
		exitErrorf(
			"Unable to get public IP for instance [%s], %v", instanceID, err)
	}
	fmt.Printf("ssh -i %s.pem ubuntu@%s\n", keyName, pubIP)

}

func getEC2Client(region string) (*ec2.EC2, error) {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return nil, err
	}
	cfg.Region = endpoints.UsEast1RegionID
	ec2Svc := ec2.New(cfg)
	return ec2Svc, nil
}

type amazonImage struct {
	ID           string
	CreationDate time.Time
}

type sortableamazonImage []*amazonImage

func (s sortableamazonImage) Len() int {
	return len(s)
}

func (s sortableamazonImage) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s sortableamazonImage) Less(i, j int) bool {
	return s[i].CreationDate.After(s[j].CreationDate)
}

func findUbuntuAMI(client *ec2.EC2) (string, error) {
	fmt.Println("running findUbuntuAMI...")
	diiFilter := ec2.Filter{
		Name:   aws.String("name"),
		Values: []string{ubuntuImageSearch},
	}
	diiFilters := []ec2.Filter{diiFilter}
	dii := ec2.DescribeImagesInput{
		Filters: diiFilters,
	}
	dir := client.DescribeImagesRequest(&dii)
	amis, err := dir.Send()
	if err != nil {
		return "", err
	}

	amisToSort := make([]*amazonImage, 0)
	for _, ami := range amis.Images {
		amiCreationDate, err := time.Parse(time.RFC3339, *ami.CreationDate)
		if err != nil {
			log.Fatal(err)
		}
		if len(ami.ProductCodes) > 0 {
			fmt.Println("Skipping image:", *ami.ImageId)
			continue
		}
		amiToAdd := &amazonImage{
			ID:           *ami.ImageId,
			CreationDate: amiCreationDate,
		}
		amisToSort = append(amisToSort, amiToAdd)
	}
	sort.Sort(sortableamazonImage(amisToSort))
	return amisToSort[0].ID, nil
}

func createSSHKeyPair(client *ec2.EC2) error {
	fmt.Println("creating createSSHKeyPair...")
	dkpi := ec2.DeleteKeyPairInput{
		KeyName: aws.String(keyName),
	}
	rkpr := client.DeleteKeyPairRequest(&dkpi)
	rkpr.Send()

	ckpi := ec2.CreateKeyPairInput{
		KeyName: aws.String(keyName),
	}
	ckp := client.CreateKeyPairRequest(&ckpi)
	createKeyPairOutput, err := ckp.Send()
	if err != nil {
		return err
	}
	writeFile(keyName+".pem", []byte(*createKeyPairOutput.KeyMaterial))
	return nil
}

func runInstance(client *ec2.EC2, ami string) (string, error) {
	fmt.Println("runInstance using AMI:", ami)
	rii := &ec2.RunInstancesInput{
		ImageId:      aws.String(ami),
		InstanceType: ec2.InstanceType(ec2Type),
		MinCount:     aws.Int64(1),
		MaxCount:     aws.Int64(1),
		KeyName:      aws.String(keyName),
	}
	rir := client.RunInstancesRequest(rii)
	reservation, err := rir.Send()
	if err != nil {
		return "", err
	}
	instanceID := reservation.Instances[0].InstanceId

	err = waitForInstanceToBeOK(*instanceID)
	if err != nil {
		return "", err
	}
	return *reservation.Instances[0].InstanceId, nil
}

func getInstancePublicIP(client *ec2.EC2, instanceID string) (string, error) {
	fmt.Println("getInstancePublicIP... instanceID: ", instanceID)
	dii := &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	}
	dir := client.DescribeInstancesRequest(dii)
	describeInstancesOutput, err := dir.Send()
	if err != nil {
		return "", err
	}
	return *describeInstancesOutput.Reservations[0].Instances[0].PublicIpAddress, nil
}

func writeFile(filename string, contents []byte) error {
	err := ioutil.WriteFile(filename, contents, 0400)
	if err != nil {
		return err
	}
	return nil
}

func exitErrorf(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, msg+"\n", args...)
	os.Exit(1)
}
