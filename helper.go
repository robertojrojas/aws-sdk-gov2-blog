package main

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

// aws-sdk-go/2.0.0-preview.2 has as in issues while waiting for resources:
// https://github.com/aws/aws-sdk-go-v2/issues/92
// I'm using aws-sdk-go v1 until this issue is resolved
func waitForInstanceToBeOK(instanceID string) error {

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	// Create new EC2 client
	client := ec2.New(sess)
	fmt.Println("waiting for instance using sdk V1 ...", instanceID)

	disi := &ec2.DescribeInstanceStatusInput{
		InstanceIds: []*string{&instanceID},
	}
	err := client.WaitUntilInstanceStatusOk(disi)
	if err != nil {
		return err
	}
	return nil
}
