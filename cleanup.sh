#!/bin/bash

INSTANCE_ID=$1
KEY_PAIR=$2

aws ec2 terminate-instances --instance-ids ${INSTANCE_ID}
aws ec2 wait instance-terminated --instance-ids ${INSTANCE_ID}

aws ec2 delete-key-pair --key-name ${KEY_PAIR}
rm -rf {$KEY_PAIR}.pem

