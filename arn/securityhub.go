package arn

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/securityhub"
	"github.com/aws/aws-sdk-go-v2/service/securityhub/types"
)

func (c *Collector) collectArnsSecurityhubExtras(ctx context.Context, cfg aws.Config) (Arns, error) {
	arns := Arns{}
	client := securityhub.NewFromConfig(cfg)
	hub, err := client.DescribeHub(ctx, &securityhub.DescribeHubInput{})
	if err != nil {
		return nil, err
	}
	if hub.SubscribedAt == nil {
		return arns, nil
	}
	a, err := New(*hub.HubArn)
	if err != nil {
		return nil, err
	}
	arns = append(arns, a)
	enabled, err := client.GetEnabledStandards(ctx, &securityhub.GetEnabledStandardsInput{})
	if err != nil {
		return nil, err
	}
	for _, s := range enabled.StandardsSubscriptions {
		a, err := New(*s.StandardsArn)
		if err != nil {
			return nil, err
		}
		arns = append(arns, a)
		var nt *string
		for {
			ctrls, err := client.DescribeStandardsControls(ctx, &securityhub.DescribeStandardsControlsInput{
				StandardsSubscriptionArn: s.StandardsSubscriptionArn,
				NextToken:                nt,
				MaxResults:               int32(100),
			})
			if err != nil {
				return nil, err
			}
			for _, ctrl := range ctrls.Controls {
				if ctrl.ControlStatus == types.ControlStatusEnabled {
					a, err := New(*ctrl.StandardsControlArn)
					if err != nil {
						return nil, err
					}
					arns = append(arns, a)
				}
			}
			nt = ctrls.NextToken
			if ctrls.NextToken == nil {
				break
			}
		}
	}
	return arns, nil
}
