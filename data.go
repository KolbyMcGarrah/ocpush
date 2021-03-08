package ocpush

import (
	"fmt"

	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

// RequestData holds all of the view data for the registered views
type RequestData struct {
	Views []*ViewData
}

// ViewData represents the data format required for push gateway: https://docs.google.com/document/d/1ZjyKiKxZV83VI9ZKAXRGKaUKK2BIWCT7oiGBKDBpjEY/edit#
type ViewData struct {
	BaseLabels baseLabel `json:"baseLabels"`
	Docstring  string    `json:"docstring"`
	Metric     metric    `json:"metric"`
}

type baseLabel struct {
	Name string `json:"__name__"`
}

type metric struct {
	Type   string  `json:"type"`
	Values []value `json:"value"`
}

type value struct {
	Label map[string]string `json:"labels"`
	Value interface{}       `json:"value"`
}

func buildRequest(rows []*view.Row, v *view.View) *ViewData {
	var reqMetric metric

	request := &ViewData{
		BaseLabels: baseLabel{
			Name: v.Name,
		},
		Docstring: v.Description,
	}

	reqMetric.Type = getType(v.Aggregation.Type)

	for i, row := range rows {
		reqMetric.Values = append(reqMetric.Values, value{
			Label: getLabels(row.Tags),
		})
		if reqMetric.Type == "histogram" {
			reqMetric.Values[i].Value = histogramValues(row, v)
		}
	}

	request.Metric = reqMetric

	return request
}

func getLabels(tags []tag.Tag) map[string]string {
	labels := make(map[string]string)
	return labels
}

func histogramValues(row *view.Row, v *view.View) map[string]int64 {
	values := make(map[string]int64)
	for i, bucket := range v.Aggregation.Buckets {
		values[fmt.Sprintf("%f", bucket)] = row.Data.(*view.DistributionData).CountPerBucket[i]
	}
	return values
}
