package awsclient

import (
	"testing"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
}

type mockRoute53 struct {
	mock.Mock
}

func (m *mockRoute53) ListHostedZonesPages(params *route53.ListHostedZonesInput, fn func(*route53.ListHostedZonesOutput, bool) bool) error {
	args := m.Called(params, fn)

	// simulate multiple pages
	out := &route53.ListHostedZonesOutput{
		HostedZones: []*route53.HostedZone{
			&route53.HostedZone{Id: aws.String("/hostedzone/ZID12341"), Name: aws.String("example1.com.")},
			&route53.HostedZone{Id: aws.String("/hostedzone/ZID12342"), Name: aws.String("example2.com.")},
		},
		IsTruncated: aws.Bool(true),
	}
	fn(out, false)

	out = &route53.ListHostedZonesOutput{
		HostedZones: []*route53.HostedZone{
			&route53.HostedZone{Id: aws.String("/hostedzone/ZID12343"), Name: aws.String("example3.com.")},
		},
		IsTruncated: aws.Bool(false),
	}
	fn(out, true)

	return args.Error(0)
}

func (m *mockRoute53) ChangeResourceRecordSets(input *route53.ChangeResourceRecordSetsInput) (*route53.ChangeResourceRecordSetsOutput, error) {
	args := m.Called(input)

	return args.Get(0).(*route53.ChangeResourceRecordSetsOutput), args.Error(1)
}

func (m *mockRoute53) ListResourceRecordSetsPages(params *route53.ListResourceRecordSetsInput, fn func(*route53.ListResourceRecordSetsOutput, bool) bool) error {
	args := m.Called(params, fn)

	// simulate multiple pages
	out := &route53.ListResourceRecordSetsOutput{
		ResourceRecordSets: []*route53.ResourceRecordSet{
			&route53.ResourceRecordSet{
				Name: aws.String("test1.example1.com."),
				AliasTarget: &route53.AliasTarget{
					DNSName:              aws.String("dsdsdf.us-east-1.elb.amazonaws.com"),
					HostedZoneId:         aws.String("Z2HXXXXXXXXXXX"),
					EvaluateTargetHealth: aws.Bool(true),
				},
			},
			&route53.ResourceRecordSet{
				Name: aws.String("test2.example1.com."),
				AliasTarget: &route53.AliasTarget{
					DNSName:              aws.String("kjskjk.us-east-1.elb.amazonaws.com"),
					HostedZoneId:         aws.String("Z2IXXXXXXXXXXX"),
					EvaluateTargetHealth: aws.Bool(true),
				},
			},
		},
		IsTruncated: aws.Bool(true),
	}
	fn(out, false)

	out = &route53.ListResourceRecordSetsOutput{
		ResourceRecordSets: []*route53.ResourceRecordSet{
			&route53.ResourceRecordSet{
				Name: aws.String("test3.example1.com."),
				AliasTarget: &route53.AliasTarget{
					DNSName:              aws.String("jdlkjs.us-east-1.elb.amazonaws.com"),
					HostedZoneId:         aws.String("Z2FXXXXXXXXXXX"),
					EvaluateTargetHealth: aws.Bool(true),
				},
			},
		},
		IsTruncated: aws.Bool(false),
	}
	fn(out, true)

	return args.Error(0)
}

func TestZones(t *testing.T) {
	c := New()
	r53 := &mockRoute53{}
	c.r53 = r53
	mockListHostedZones(r53, nil)

	zones, err := c.Zones()

	assert.Nil(t, err, "error should be nil")
	assert.Equal(t, 3, len(zones), "len(zones) should be 3")

	assert.Equal(t, "/hostedzone/ZID12341", zones[0].ID)
	assert.Equal(t, "example1.com.", zones[0].Name)
	assert.Equal(t, "/hostedzone/ZID12342", zones[1].ID)
	assert.Equal(t, "example2.com.", zones[1].Name)
	assert.Equal(t, "/hostedzone/ZID12343", zones[2].ID)
	assert.Equal(t, "example3.com.", zones[2].Name)
}

func TestZonesWithBadCredentials(t *testing.T) {
	c := New()
	r53 := &mockRoute53{}
	c.r53 = r53
	mockListHostedZones(r53, credentials.ErrNoValidProvidersFoundInChain)

	_, err := c.Zones()
	assert.Equal(t, ErrInvalidAWSCredentials, err)
}

func TestFindZone(t *testing.T) {
	c := New()
	r53 := &mockRoute53{}
	c.r53 = r53
	mockListHostedZones(r53, nil)

	zoneNames := []string{"example2.com.", "example2.com"}
	for _, name := range zoneNames {
		zone, err := c.FindZone(name)
		assert.Nil(t, err, "error should be nil")
		assert.Equal(t, "/hostedzone/ZID12342", zone.ID)
	}
}

func TestFindZoneNoExist(t *testing.T) {
	c := New()
	r53 := &mockRoute53{}
	c.r53 = r53
	mockListHostedZones(r53, nil)

	name := "example-notfound.com"
	zone, err := c.FindZone(name)

	assert.Equal(t, ErrZoneNotFound, err, "error should be ErrZoneNotFound")
	assert.Nil(t, zone, "zone should be nil")
}

func TestFindRecord(t *testing.T) {
	c := New()
	r53 := &mockRoute53{}
	c.r53 = r53
	mockListResourceRecordSets(r53, nil)

	zone := &Zone{
		ID:   "/hostedzone/ZID12341",
		Name: "example1.com.",
	}

	aliases := []string{"test2", "test2.example1.com.", "test2.example1.com"}
	for _, alias := range aliases {
		rec, err := c.FindRecord(zone, alias)
		assert.NotNil(t, rec)
		assert.Nil(t, err)
		assert.Equal(t, "test2.example1.com.", rec.Name)
		assert.Equal(t, "kjskjk.us-east-1.elb.amazonaws.com", rec.DNSName)
		assert.Equal(t, "Z2IXXXXXXXXXXX", rec.HostedZoneID)
	}
}

func TestSetAlias(t *testing.T) {
	c := New()
	r53 := &mockRoute53{}
	c.r53 = r53

	output := &route53.ChangeResourceRecordSetsOutput{
		ChangeInfo: &route53.ChangeInfo{
			Id:          aws.String("11111"),
			Status:      aws.String(route53.ChangeStatusPending),
			SubmittedAt: aws.Time(time.Now()),
		},
	}

	r53.Mock.On(
		"ChangeResourceRecordSets",
		mock.AnythingOfType("*route53.ChangeResourceRecordSetsInput"),
	).Return(output, nil)

	zone := &Zone{ID: "/hostedzone/ZID12342", Name: "example2.com."}

	// handle different alias formats
	aliases := []string{"test.example2.com.", "test", "test.example2.com"}
	for _, alias := range aliases {
		change, err := c.SetAlias(zone, "Z3DZX7HGU9N41H", "aerfflakjdfljadlkfjal-77828384.us-east-1.elb.amazonaws.com", alias)
		assert.Nil(t, err, "error should be nil")
		assert.Equal(t, route53.ChangeStatusPending, aws.StringValue(change.Status), "status should be pending")
	}
}

func TestRemoveAlias(t *testing.T) {
	c := New()
	r53 := &mockRoute53{}
	c.r53 = r53
	mockListResourceRecordSets(r53, nil)

	output := &route53.ChangeResourceRecordSetsOutput{
		ChangeInfo: &route53.ChangeInfo{
			Id:          aws.String("11111"),
			Status:      aws.String(route53.ChangeStatusPending),
			SubmittedAt: aws.Time(time.Now()),
		},
	}

	r53.Mock.On(
		"ChangeResourceRecordSets",
		mock.AnythingOfType("*route53.ChangeResourceRecordSetsInput"),
	).Return(output, nil)

	zone := &Zone{ID: "/hostedzone/ZID12341", Name: "example1.com."}

	// handle different alias formats
	aliases := []string{"test2.example1.com.", "test2", "test2.example1.com"}
	for _, alias := range aliases {
		change, err := c.RemoveAlias(zone, alias)
		assert.Nil(t, err, "error should be nil")
		assert.Equal(t, route53.ChangeStatusPending, aws.StringValue(change.Status), "status should be pending")
	}
}

func mockListHostedZones(m *mockRoute53, returnParams ...interface{}) {
	m.Mock.On(
		"ListHostedZonesPages",
		mock.AnythingOfType("*route53.ListHostedZonesInput"),
		mock.AnythingOfType("func(*route53.ListHostedZonesOutput, bool) bool"),
	).Return(returnParams...)
}

func mockListResourceRecordSets(m *mockRoute53, returnParams ...interface{}) {
	m.Mock.On(
		"ListResourceRecordSetsPages",
		mock.AnythingOfType("*route53.ListResourceRecordSetsInput"),
		mock.AnythingOfType("func(*route53.ListResourceRecordSetsOutput, bool) bool"),
	).Return(returnParams...)
}
