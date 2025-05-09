package aws

import (
	"io"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/awslabs/k8s-cloudwatch-adapter/pkg/apis/metrics/v1alpha1"

	"k8s.io/klog"
)

// GetLocalRegion gets the region ID from the instance metadata using IMDSv2, falling back to AWS_REGION env.
func GetLocalRegion() string {
	// First, get a token for IMDSv2
	tokenReq, err := http.NewRequest("PUT", "http://169.254.169.254/latest/api/token", nil)
	if err != nil {
		klog.Errorf("unable to create token request, %v", err)
		return os.Getenv("AWS_REGION")
	}
	tokenReq.Header.Set("X-aws-ec2-metadata-token-ttl-seconds", "21600") // 6 hours

	client := &http.Client{Timeout: 5 * time.Second}
	tokenResp, err := client.Do(tokenReq)
	if err != nil {
		klog.Errorf("unable to get IMDSv2 token, %v", err)
		return os.Getenv("AWS_REGION")
	}
	defer tokenResp.Body.Close()

	token, err := io.ReadAll(tokenResp.Body)
	if err != nil {
		klog.Errorf("cannot read token response, %v", err)
		return os.Getenv("AWS_REGION")
	}

	// Now use the token to get the availability zone
	azReq, err := http.NewRequest("GET", "http://169.254.169.254/latest/meta-data/placement/availability-zone", nil)
	if err != nil {
		klog.Errorf("unable to create AZ request, %v", err)
		return os.Getenv("AWS_REGION")
	}
	azReq.Header.Set("X-aws-ec2-metadata-token", string(token))

	azResp, err := client.Do(azReq)
	if err != nil {
		klog.Errorf("unable to get current region information, %v", err)
		return os.Getenv("AWS_REGION")
	}
	defer azResp.Body.Close()

	body, err := io.ReadAll(azResp.Body)
	if err != nil {
		klog.Errorf("cannot read response from instance metadata, %v", err)
		return os.Getenv("AWS_REGION")
	}

	// strip the last character from AZ to get region ID
	return string(body[0 : len(body)-1])
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
