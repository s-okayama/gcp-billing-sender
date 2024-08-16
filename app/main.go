package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/zorkian/go-datadog-api"
	"google.golang.org/api/iterator"
)

type BillingData struct {
	Month       string
	Description string
	Total       float64
	TotalExact  float64
}

func queryBigQuery(ctx context.Context, client *bigquery.Client, projectID, dataset string) ([]BillingData, error) {
	query := fmt.Sprintf(`
SELECT
  invoice.month,
  service.description,
  SUM(cost) + SUM(IFNULL((
    SELECT
     SUM(credit.amount)
   FROM
     UNNEST(credits) AS credit
  ), 0)) AS total,
  SUM(CAST(cost AS BIGNUMERIC)) + SUM(IFNULL((
    SELECT
     SUM(CAST(credit.amount AS BIGNUMERIC))
   FROM
     UNNEST(credits) AS credit
  ), 0)) AS total_exact
FROM
  %s.billing.%s
WHERE invoice.month LIKE FORMAT_TIMESTAMP('%%Y%%m', CURRENT_TIMESTAMP())
GROUP BY 1, 2
ORDER BY 1 ASC, 2 ASC;
`, projectID, dataset)

	it, err := client.Query(query).Read(ctx)
	if err != nil {
		return nil, err
	}

	var results []BillingData
	for {
		var row BillingData
		err := it.Next(&row)
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		results = append(results, row)
	}

	return results, nil
}

func sendToDatadog(data []BillingData) error {
	apiKey := os.Getenv("DATADOG_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("DATADOG_API_KEY environment variable is not set")
	}

	client := datadog.NewClient(apiKey, "")

	for _, d := range data {
		metric := datadog.Metric{
			Metric: stringPtr("gcp.billing.total"),
			Points: []datadog.DataPoint{
				{float64Ptr(float64(time.Now().Unix())), float64Ptr(d.Total)},
			},
			Tags: []string{
				fmt.Sprintf("month:%s", d.Month),
				fmt.Sprintf("description:%s", d.Description),
			},
		}
		err := client.PostMetrics([]datadog.Metric{metric})
		if err != nil {
			return err
		}
	}

	return nil
}

func stringPtr(s string) *string {
	return &s
}

func float64Ptr(f float64) *float64 {
	return &f
}

func main() {
	ctx := context.Background()
	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		log.Fatalf("GCP_PROJECT_ID environment variable is not set")
	}

	dataset := os.Getenv("BIGQUERY_DATASET")
	if dataset == "" {
		log.Fatalf("DATASET environment variable is not set")
	}

	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to create BigQuery client: %v", err)
	}
	defer client.Close()

	data, err := queryBigQuery(ctx, client, projectID, dataset)
	if err != nil {
		log.Fatalf("Failed to query BigQuery: %v", err)
	}

	err = sendToDatadog(data)
	if err != nil {
		log.Fatalf("Failed to send data to Datadog: %v", err)
	}

	log.Println("Data successfully sent to Datadog")
}
