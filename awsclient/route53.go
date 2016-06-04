package awsclient

import (
	"errors"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
)

var (
	ErrZoneNotFound   = errors.New("Zone does not exist.")
	ErrRecordNotFound = errors.New("Record does not exist.")
)

type Route53er interface {
	ListHostedZonesPages(*route53.ListHostedZonesInput, func(*route53.ListHostedZonesOutput, bool) bool) error
	ChangeResourceRecordSets(input *route53.ChangeResourceRecordSetsInput) (*route53.ChangeResourceRecordSetsOutput, error)
	ListResourceRecordSetsPages(*route53.ListResourceRecordSetsInput, func(*route53.ListResourceRecordSetsOutput, bool) bool) error
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
		return nil, checkError(err)
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
		return nil, checkError(err)
	}

	if rec == nil {
		return nil, ErrRecordNotFound
	}

	return rec, nil
}

func (c *AWSClient) SetAlias(zone *Zone, hzid, elbDnsName, alias string) (*route53.ChangeInfo, error) {
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
		return nil, err
	}

	return out.ChangeInfo, nil
}

func (c *AWSClient) RemoveAlias(zone *Zone, alias string) (*route53.ChangeInfo, error) {
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
		return nil, err
	}

	return out.ChangeInfo, nil
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
