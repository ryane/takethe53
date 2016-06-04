package awsclient

import (
	"errors"

	"github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/aws/aws-sdk-go/service/route53"
)

type AWSClient struct {
	r53 Route53er
	elb ELBer
}

var (
	ErrInvalidAWSCredentials = errors.New("Invalid AWS Credentials. Please see https://github.com/aws/aws-sdk-go#configuring-credentials.")
)

func New() *AWSClient {
	sess := session.New()
	sess.Handlers.Send.PushFront(func(r *request.Request) {
		logrus.WithFields(logrus.Fields{
			"service": r.ClientInfo.ServiceName,
			"op":      r.Operation.Name,
			"method":  r.Operation.HTTPMethod,
			"path":    r.Operation.HTTPPath,
			"type":    "aws",
			"params":  r.Params,
		}).Debug("request.aws: ", r.Operation.Name)
	})

	awsConfig := &aws.Config{
		CredentialsChainVerboseErrors: aws.Bool(true),
	}

	r53er := route53.New(sess, awsConfig)
	elber := elb.New(sess, awsConfig)

	return &AWSClient{r53: r53er, elb: elber}
}

func checkAWSError(err error) error {
	awserr := err.(awserr.Error)
	logrus.WithFields(logrus.Fields{
		"type":  "aws",
		"error": awserr,
		"code":  awserr.Code(),
	}).Debug("error.aws")

	switch awserr {
	case credentials.ErrNoValidProvidersFoundInChain:
		return ErrInvalidAWSCredentials
	}

	return err
}
