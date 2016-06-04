package awsclient

import (
	"errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elb"
	"strings"
)

type ELBer interface {
	DescribeLoadBalancersPages(input *elb.DescribeLoadBalancersInput, fn func(p *elb.DescribeLoadBalancersOutput, lastPage bool) (shouldContinue bool)) error
}

type LoadBalancer struct {
	Name         string
	HostedZoneID string
}

var ErrELBNotFound = errors.New("ELB does not exist.")

func (c *AWSClient) LoadBalancers() ([]*LoadBalancer, error) {
	params := &elb.DescribeLoadBalancersInput{PageSize: aws.Int64(400)}

	var lbs []*LoadBalancer
	err := c.elb.DescribeLoadBalancersPages(params, func(o *elb.DescribeLoadBalancersOutput, lastPage bool) bool {
		for _, lbd := range o.LoadBalancerDescriptions {
			lbs = append(lbs, &LoadBalancer{
				Name:         aws.StringValue(lbd.DNSName),
				HostedZoneID: aws.StringValue(lbd.CanonicalHostedZoneNameID),
			})
		}
		return !lastPage
	})

	if err != nil {
		return nil, checkError(err)
	}

	return lbs, nil
}

func (c *AWSClient) FindLoadBalancer(dnsName string) (*LoadBalancer, error) {
	lbs, err := c.LoadBalancers()
	if err != nil {
		return nil, err
	}

	for _, lb := range lbs {
		if strings.EqualFold(dnsName, lb.Name) {
			return lb, nil
		}
	}

	return nil, ErrELBNotFound
}
