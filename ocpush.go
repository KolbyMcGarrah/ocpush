package ocpush

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

// PushExporter holds the job name and meter to be used to
// push metrics to the pushgateway.
type PushExporter struct {
	Meter     view.Meter
	namespace string
	jobName   string
	instance  string
	views     []*view.View
	pushAddr  string
	pushPort  string
	isTest    bool
}

// NewPushExporter creates a new PushExporter and starts the Meter
// so that it can collect metrics
// JobName defines the name that will be associated with the metrics
// in push gateway
func NewPushExporter(isTest bool, namespace, pushAddr, pushPort, jobName string) *PushExporter {
	var pe = &PushExporter{
		Meter:     view.NewMeter(),
		namespace: namespace,
		pushAddr:  pushAddr,
		pushPort:  pushPort,
		jobName:   jobName,
		isTest:    isTest,
	}
	pe.Meter.Start()
	return pe
}

// SetInstance sets the name of the instance for the metrics collection
func (pe *PushExporter) SetInstance(instance string) {
	pe.instance = instance
}

// RegisterViews registers all of the opencensus views with the Meter
// So that they can begin recording metrics.
func (pe *PushExporter) RegisterViews(views ...*view.View) error {
	// add the view names to the view name slice to be able to export later
	for _, view := range views {
		pe.views = append(pe.views, view)
	}
	return pe.Meter.Register(views...)
}

// Record records the measurement to the Meter so that it can be exported
// Tags will be grabbed from the context
// The second argument is a `[]Measurement`.
func (pe *PushExporter) Record(ctx context.Context, measurement []stats.Measurement, attachments map[string]interface{}) {
	pe.Meter.Record(tag.FromContext(ctx), measurement, attachments)
}

// PushMetrics extracts all of the data from the views and pushes them to the pushgateway
// Errors are being returned on a channel in case we encounter multiple
func (pe *PushExporter) PushMetrics() {
	client := &http.Client{}
	// push metrics for each view registered to the Meter
	for _, view := range pe.views {
		metricName := fmt.Sprint(pe.namespace, "_", view.Name)
		helpString := fmt.Sprint("#HELP ", metricName, " ", view.Description, "\n")
		typeString := fmt.Sprint("#TYPE ", metricName, " ", getType(view.Aggregation.Type), "\n")
		reqData := fmt.Sprint(helpString, typeString)
		rows, err := pe.Meter.RetrieveData(view.Name)
		for _, row := range rows {
			reqData = fmt.Sprint(reqData, metricName, formatRowData(metricName, row, view), "\n")
			if err != nil || len(rows) < 1 {
				continue
			}

		}
		fmt.Println(reqData)
		req, err := http.NewRequest(http.MethodPost, pe.buildURLString(), bytes.NewBuffer([]byte(reqData)))
		if err != nil {
		}
		req.Header.Set("Content-Type", "plain/text; charset=utf-8")
		resp, err := client.Do(req)
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println(err)
		}
		bodyString := string(bodyBytes)
		fmt.Println(bodyString)
	}
}

func (pe *PushExporter) buildURLString() string {
	var url string
	url = fmt.Sprintf("%s%s/metrics", pe.pushAddr, pe.pushPort)
	if pe.jobName != "" {
		url = fmt.Sprintf("%s/job/%s", url, pe.jobName)
	}
	if pe.instance != "" {
		url = fmt.Sprintf("%s/instance/%s", url, pe.instance)
	}
	return url
}

func formatRowData(metricName string, row *view.Row, v *view.View) string {
	var formattedData = "{"
	for i, tag := range row.Tags {
		if i == 0 {
			formattedData = fmt.Sprint(formattedData, tag.Key.Name(), `="`, tag.Value, `"`)
			continue
		}
		formattedData = fmt.Sprint(formattedData, ",", tag.Key.Name(), `="`, tag.Value, `"`)
	}
	switch v.Aggregation.Type {
	case view.AggTypeCount:
		formattedData = fmt.Sprint(formattedData, "} ", row.Data.(*view.CountData).Value)
	case view.AggTypeSum:
		formattedData = fmt.Sprint(formattedData, "} ", row.Data.(*view.SumData).Value)
	case view.AggTypeLastValue:
		formattedData = fmt.Sprint(formattedData, "} ", row.Data.(*view.LastValueData).Value)
	// TODO Set up bucket distributions
	case view.AggTypeDistribution:
		rowData := formattedData
		for i, bucket := range v.Aggregation.Buckets {
			if i == 0 {
				formattedData = fmt.Sprint(formattedData, ", quantile=", fmt.Sprintf(`"%f"}`, bucket), row.Data.(*view.DistributionData).CountPerBucket[i], "\n")
			} else {
				formattedData = fmt.Sprint(formattedData, metricName, rowData, ", quantile=", fmt.Sprintf(`"%f"}`, bucket), row.Data.(*view.DistributionData).CountPerBucket[i], "\n")
			}
		}
		formattedData = fmt.Sprint(formattedData, metricName, "_sum ", row.Data.(*view.DistributionData).Sum(), "\n")
		formattedData = fmt.Sprint(formattedData, metricName, "_count ", row.Data.(*view.DistributionData).Count, "\n")
	default:
		formattedData = fmt.Sprint(formattedData, "} ")
	}

	return formattedData
}

func getType(aggType view.AggType) string {
	var returnType string
	switch aggType {
	case view.AggTypeCount:
		returnType = "counter"
	case view.AggTypeSum:
		returnType = "summary"
	case view.AggTypeLastValue:
		returnType = "gauge"
	case view.AggTypeDistribution:
		returnType = "histogram"
	default:
		returnType = "untyped"
	}
	return returnType
}
