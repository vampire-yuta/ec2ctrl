package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"

	"github.com/olekukonko/tablewriter"
)

var (
	argProfile   = flag.String("profile", "", "Profile 名を指定.")
	argRegion    = flag.String("region", "ap-northeast-1", "Region 名を指定.")
	argInstances = flag.String("instances", "", "Instance ID 又は Instance Tag 名を指定.")
	argStart     = flag.Bool("start", false, "Instance を起動.")
	argStop      = flag.Bool("stop", false, "Instance を停止.")
)

func outputTbl(data [][]string) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"tag:Name", "InstanceId", "InstanceType", "AZ", "PrivateIp", "PublicIp", "Status"})

	for _, value := range data {
		table.Append(value)
	}
	table.Render()
}

func awsEc2Client(profile string, region string) *ec2.EC2 {
	var config aws.Config
	if profile != "" {
		creds := credentials.NewSharedCredentials("", profile)
		config = aws.Config{Region: aws.String(region), Credentials: creds}
	} else {
		config = aws.Config{Region: aws.String(region)}
	}
	sess := session.New(&config)
	ec2Client := ec2.New(sess)
	return ec2Client
}

func listInstances(ec2Client *ec2.EC2, instances []*string) {
	params := &ec2.DescribeInstancesInput{
		InstanceIds: instances,
	}
	res, err := ec2Client.DescribeInstances(params)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	allInstances := [][]string{}
	for _, r := range res.Reservations {
		for _, i := range r.Instances {
			var tag_name string
			for _, t := range i.Tags {
				if *t.Key == "Name" {
					tag_name = *t.Value
				}
			}
			if i.PublicIpAddress == nil {
				i.PublicIpAddress = aws.String("Not assignment")
			}
			instance := []string{
				tag_name,
				*i.InstanceId,
				*i.InstanceType,
				*i.Placement.AvailabilityZone,
				*i.PrivateIpAddress,
				*i.PublicIpAddress,
				*i.State.Name,
			}
			allInstances = append(allInstances, instance)
		}
	}
	outputTbl(allInstances)
}

func startInstances(ec2Client *ec2.EC2, instances []*string) {
	params := &ec2.StartInstancesInput{
		InstanceIds: instances,
	}
	result, err := ec2Client.StartInstances(params)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	for _, r := range result.StartingInstances {
		fmt.Printf("%s を起動しました.\n", *r.InstanceId)
	}
}

func stopInstances(ec2Client *ec2.EC2, instances []*string) {
	params := &ec2.StopInstancesInput{
		InstanceIds: instances,
	}
	// fmt.Println(params)
	result, err := ec2Client.StopInstances(params)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	for _, r := range result.StoppingInstances {
		fmt.Printf("%s を停止しました.\n", *r.InstanceId)
	}
}

func ctrlInstances(ec2Client *ec2.EC2, instances []*string, operation string) {
	listInstances(ec2Client, instances)

	fmt.Print("上記のインスタンスを操作しますか?(y/n): ")
	var stdin string
	fmt.Scan(&stdin)
	switch stdin {
	case "y", "Y":
		switch operation {
		case "start":
			fmt.Println("EC2 を起動します.")
			startInstances(ec2Client, instances)
		case "stop":
			fmt.Println("EC2 を停止します.")
			stopInstances(ec2Client, instances)
		}
	case "n", "N":
		fmt.Println("処理を停止します.")
		os.Exit(0)
	default:
		fmt.Println("処理を停止します.")
		os.Exit(0)
	}
}

func getInstanceIds(ec2Client *ec2.EC2, instances string) []*string {
	splitedInstances := strings.Split(instances, ",")
	res, err := ec2Client.DescribeInstances(nil)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	var instanceIds []*string
	for _, s := range splitedInstances {
		for _, r := range res.Reservations {
			for _, i := range r.Instances {
				for _, t := range i.Tags {
					if *t.Key == "Name" {
						if *t.Value == s {
							instanceIds = append(instanceIds, aws.String(*i.InstanceId))
						}
					}
				}
				if *i.InstanceId == s {
					instanceIds = append(instanceIds, aws.String(*i.InstanceId))
				}
			}
		}
	}
	return instanceIds
}

func main() {
	flag.Parse()
	//if *argProfile == "" {
	//   fmt.Println("`-profile` オプションを指定して下さい.")
	//    os.Exit(1)
	//}

	ec2Client := awsEc2Client(*argProfile, *argRegion)
	var instances []*string
	if *argInstances != "" {
		instances = getInstanceIds(ec2Client, *argInstances)
		if *argStart {
			ctrlInstances(ec2Client, instances, "start")
		} else if *argStop {
			ctrlInstances(ec2Client, instances, "stop")
		} else {
			fmt.Println("`-start` 又は `-stop` オプションを指定して下さい.")
			os.Exit(1)
		}
	} else {
		listInstances(ec2Client, nil)
	}
}
