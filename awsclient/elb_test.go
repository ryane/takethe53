package awsclient

import (
	"github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
}

type mockELB struct {
	mock.Mock
}

func (m *mockELB) DescribeLoadBalancersPages(params *elb.DescribeLoadBalancersInput, fn func(*elb.DescribeLoadBalancersOutput, bool) bool) error {
	args := m.Called(params, fn)

	// simulate multiple pages
	out := &elb.DescribeLoadBalancersOutput{
		LoadBalancerDescriptions: []*elb.LoadBalancerDescription{
			&elb.LoadBalancerDescription{
				DNSName:                   aws.String("ab4xxxxxxxxxxxxxxxxxxxxxxxxxxxxx-xxxxxxxxx.us-east-1.elb.amazonaws.com"),
				CanonicalHostedZoneNameID: aws.String("Z3DXXXXXXXXXXX"),
			},
			&elb.LoadBalancerDescription{
				DNSName:                   aws.String("afexxxxxxxxxxxxxxxxxxxxxxxxxxxxx-xxxxxxxxx.us-east-1.elb.amazonaws.com"),
				CanonicalHostedZoneNameID: aws.String("Z2HXXXXXXXXXXX"),
			},
		},
		NextMarker: aws.String("marker"),
	}
	fn(out, false)

	out = &elb.DescribeLoadBalancersOutput{
		LoadBalancerDescriptions: []*elb.LoadBalancerDescription{
			&elb.LoadBalancerDescription{
				DNSName:                   aws.String("a2bxxxxxxxxxxxxxxxxxxxxxxxxxxxxx-xxxxxxxxx.us-east-1.elb.amazonaws.com"),
				CanonicalHostedZoneNameID: aws.String("Z8IXXXXXXXXXXX"),
			},
		},
		NextMarker: aws.String(""),
	}
	fn(out, true)

	return args.Error(0)
}

func TestLoadBalancers(t *testing.T) {
	c := New()
	elber := &mockELB{}
	c.elb = elber
	mockDescribeLoadBalancers(elber, nil)

	lbs, err := c.LoadBalancers()

	assert.Nil(t, err, "error should be nil")
	assert.Equal(t, 3, len(lbs), "len(lbs) should be 2")

	assert.Equal(t, "ab4xxxxxxxxxxxxxxxxxxxxxxxxxxxxx-xxxxxxxxx.us-east-1.elb.amazonaws.com", lbs[0].Name)
	assert.Equal(t, "Z3DXXXXXXXXXXX", lbs[0].HostedZoneID)
	assert.Equal(t, "afexxxxxxxxxxxxxxxxxxxxxxxxxxxxx-xxxxxxxxx.us-east-1.elb.amazonaws.com", lbs[1].Name)
	assert.Equal(t, "Z2HXXXXXXXXXXX", lbs[1].HostedZoneID)
	assert.Equal(t, "a2bxxxxxxxxxxxxxxxxxxxxxxxxxxxxx-xxxxxxxxx.us-east-1.elb.amazonaws.com", lbs[2].Name)
	assert.Equal(t, "Z8IXXXXXXXXXXX", lbs[2].HostedZoneID)
}

func TestLoadBalancersWithBadCredentials(t *testing.T) {
	c := New()
	elber := &mockELB{}
	c.elb = elber
	mockDescribeLoadBalancers(elber, credentials.ErrNoValidProvidersFoundInChain)

	_, err := c.LoadBalancers()
	assert.Equal(t, ErrInvalidAWSCredentials, err)
}

func TestFindLoadBalancer(t *testing.T) {
	c := New()
	elber := &mockELB{}
	c.elb = elber
	mockDescribeLoadBalancers(elber, nil)

	name := "afexxxxxxxxxxxxxxxxxxxxxxxxxxxxx-xxxxxxxxx.us-east-1.elb.amazonaws.com"
	lb, err := c.FindLoadBalancer(name)

	assert.Nil(t, err, "error should be nil")
	assert.Equal(t, "Z2HXXXXXXXXXXX", lb.HostedZoneID, "shou")
}

func TestFindLoadBalancerNoExist(t *testing.T) {
	c := New()
	elber := &mockELB{}
	c.elb = elber
	mockDescribeLoadBalancers(elber, nil)

	name := "abnxxxxxxxxxxxxxxxxxxxxxxxxxxxxx-xxxxxxxxx.us-east-1.elb.amazonaws.com"
	lb, err := c.FindLoadBalancer(name)

	assert.Equal(t, ErrELBNotFound, err, "error should be ErrELBNotFound")
	assert.Nil(t, lb, "load balancer should be nil")
}

func mockDescribeLoadBalancers(m *mockELB, returnParams ...interface{}) {
	m.Mock.On(
		"DescribeLoadBalancersPages",
		mock.AnythingOfType("*elb.DescribeLoadBalancersInput"),
		mock.AnythingOfType("func(*elb.DescribeLoadBalancersOutput, bool) bool"),
	).Return(returnParams...)
}
