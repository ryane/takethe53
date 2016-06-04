package awsclient

import (
	"errors"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/route53"
)

var (
	ErrZoneNotFound   = errors.New("Zone does not exist.")
	ErrRecordNotFound = errors.New("Record does not exist.")
	ErrChangeNotFound = errors.New("Change does not exist.")
)

const (
	ChangeStatusPending = "PENDING"
	ChangeStatusInSync  = "INSYNC"
)

type Route53er interface {
	ListHostedZonesPages(*route53.ListHostedZonesInput, func(*route53.ListHostedZonesOutput, bool) bool) error
	ChangeResourceRecordSets(input *route53.ChangeResourceRecordSetsInput) (*route53.ChangeResourceRecordSetsOutput, error)
	ListResourceRecordSetsPages(*route53.ListResourceRecordSetsInput, func(*route53.ListResourceRecordSetsOutput, bool) bool) error
	GetChange(*route53.GetChangeInput) (*route53.GetChangeOutput, error)
}

type Zone struct {
	ID   string
	Name string
}

type Record struct {
	Name                 string
	DNSName              string
	HostedZoneID         string
	EvaluateTargetHealth bool
}

type ChangeStatus struct {
	ID          string
	Status      string
	SubmittedAt time.Time
	Comment     string
}

func (c *AWSClient) Zones() ([]*Zone, error) {
	params := &route53.ListHostedZonesInput{MaxItems: aws.String("100")}

	var zones []*Zone
	err := c.r53.ListHostedZonesPages(params, func(o *route53.ListHostedZonesOutput, lastPage bool) bool {
		for _, hz := range o.HostedZones {
			zones = append(zones, &Zone{
				Name: aws.StringValue(hz.Name),
				ID:   aws.StringValue(hz.Id),
			})
		}
		return !lastPage
	})

	if err != nil {
		return nil, checkAWSError(err)
	}

	return zones, nil
}

func (c *AWSClient) FindZone(name string) (*Zone, error) {
	zoneName := name
	if !strings.HasSuffix(zoneName, ".") {
		zoneName += "."
	}

	zones, err := c.Zones()
	if err != nil {
		return nil, err
	}

	for _, z := range zones {
		if strings.EqualFold(zoneName, z.Name) {
			return z, nil
		}
	}

	return nil, ErrZoneNotFound
}

func (c *AWSClient) FindRecord(zone *Zone, alias string) (*Record, error) {
	aliasDnsName := aliasDnsName(alias, zone)

	params := &route53.ListResourceRecordSetsInput{
		HostedZoneId: aws.String(zone.ID),
		MaxItems:     aws.String("100"),
	}

	var rec *Record
	err := c.r53.ListResourceRecordSetsPages(params, func(o *route53.ListResourceRecordSetsOutput, lastPage bool) bool {
		for _, rrs := range o.ResourceRecordSets {
			if strings.EqualFold(aws.StringValue(rrs.Name), aliasDnsName) {
				rec = &Record{
					Name:                 aws.StringValue(rrs.Name),
					DNSName:              aws.StringValue(rrs.AliasTarget.DNSName),
					HostedZoneID:         aws.StringValue(rrs.AliasTarget.HostedZoneId),
					EvaluateTargetHealth: aws.BoolValue(rrs.AliasTarget.EvaluateTargetHealth),
				}
				break
			}
		}
		return !lastPage
	})

	if err != nil {
		return nil, checkAWSError(err)
	}

	if rec == nil {
		return nil, ErrRecordNotFound
	}

	return rec, nil
}

func (c *AWSClient) SetAlias(zone *Zone, hzid, elbDnsName, alias string) (*ChangeStatus, error) {
	aliasDnsName := aliasDnsName(alias, zone)
	params := &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(zone.ID),
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{
				{
					Action: aws.String("UPSERT"),
					ResourceRecordSet: &route53.ResourceRecordSet{
						Name: aws.String(aliasDnsName),
						Type: aws.String("A"),
						AliasTarget: &route53.AliasTarget{
							HostedZoneId:         aws.String(hzid),
							DNSName:              aws.String(elbDnsName),
							EvaluateTargetHealth: aws.Bool(true),
						},
					},
				},
			},
		},
	}
	out, err := c.r53.ChangeResourceRecordSets(params)
	if err != nil {
		return nil, checkAWSError(err)
	}

	return changeInfoToChangeStatus(out.ChangeInfo), nil
}

func (c *AWSClient) RemoveAlias(zone *Zone, alias string) (*ChangeStatus, error) {
	rec, err := c.FindRecord(zone, alias)
	if err != nil {
		return nil, err
	}

	params := &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(zone.ID),
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{
				{
					Action: aws.String(route53.ChangeActionDelete),
					ResourceRecordSet: &route53.ResourceRecordSet{
						Name: aws.String(rec.Name),
						Type: aws.String("A"),
						AliasTarget: &route53.AliasTarget{
							DNSName:              aws.String(rec.DNSName),
							HostedZoneId:         aws.String(rec.HostedZoneID),
							EvaluateTargetHealth: aws.Bool(rec.EvaluateTargetHealth),
						},
					},
				},
			},
		},
	}
	out, err := c.r53.ChangeResourceRecordSets(params)
	if err != nil {
		return nil, checkAWSError(err)
	}

	return changeInfoToChangeStatus(out.ChangeInfo), nil
}

func (c *AWSClient) GetChangeStatus(id string) (*ChangeStatus, error) {
	params := &route53.GetChangeInput{
		Id: aws.String(id),
	}

	output, err := c.r53.GetChange(params)
	if err != nil {
		if awserr, ok := err.(awserr.Error); ok && awserr.Code() == "NoSuchChange" {
			return nil, ErrChangeNotFound
		} else {
			return nil, checkAWSError(err)
		}
	}

	return changeInfoToChangeStatus(output.ChangeInfo), nil
}

func aliasDnsName(alias string, zone *Zone) string {
	aliasDnsName := alias

	if !strings.HasSuffix(aliasDnsName, ".") {
		aliasDnsName += "."
	}

	if !strings.HasSuffix(aliasDnsName, zone.Name) {
		aliasDnsName += zone.Name
	}

	return aliasDnsName
}

func changeInfoToChangeStatus(ci *route53.ChangeInfo) *ChangeStatus {
	return &ChangeStatus{
		ID:          aws.StringValue(ci.Id),
		Status:      aws.StringValue(ci.Status),
		SubmittedAt: aws.TimeValue(ci.SubmittedAt),
		Comment:     aws.StringValue(ci.Comment),
	}
}
