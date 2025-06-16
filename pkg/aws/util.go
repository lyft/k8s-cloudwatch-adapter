package aws

import (
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/awslabs/k8s-cloudwatch-adapter/pkg/apis/metrics/v1alpha1"
)

// GetLocalRegion gets the region ID from the instance metadata using IMDSv2, falling back to AWS_REGION env.
func GetLocalRegion() string {
	return os.Getenv("AWS_REGION")
}

func toCloudWatchQuery(externalMetric *v1alpha1.ExternalMetric) cloudwatch.GetMetricDataInput {
	queries := externalMetric.Spec.Queries

	cwMetricQueries := make([]*cloudwatch.MetricDataQuery, len(queries))
	for i, q := range queries {
		q := q
		returnData := &q.ReturnData
		mdq := &cloudwatch.MetricDataQuery{
			Id:         &q.ID,
			Label:      &q.Label,
			ReturnData: *returnData,
		}

		if len(q.Expression) == 0 {
			dimensions := make([]*cloudwatch.Dimension, len(q.MetricStat.Metric.Dimensions))
			for j := range q.MetricStat.Metric.Dimensions {
				dimensions[j] = &cloudwatch.Dimension{
					Name:  &q.MetricStat.Metric.Dimensions[j].Name,
					Value: &q.MetricStat.Metric.Dimensions[j].Value,
				}
			}

			metric := &cloudwatch.Metric{
				Dimensions: dimensions,
				MetricName: &q.MetricStat.Metric.MetricName,
				Namespace:  &q.MetricStat.Metric.Namespace,
			}

			mdq.MetricStat = &cloudwatch.MetricStat{
				Metric: metric,
				Period: &q.MetricStat.Period,
				Stat:   &q.MetricStat.Stat,
				Unit:   aws.String(q.MetricStat.Unit),
			}
		} else {
			mdq.Expression = &q.Expression
		}

		cwMetricQueries[i] = mdq
	}
	cwQuery := cloudwatch.GetMetricDataInput{
		MetricDataQueries: cwMetricQueries,
	}

	return cwQuery
}
