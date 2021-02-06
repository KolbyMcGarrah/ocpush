package ocpush

import (
	"bytes"
	"context"
	"fmt"
	"net/http"

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
}

// NewPushExporter creates a new PushExporter and starts the Meter
// so that it can collect metrics
// JobName defines the name that will be associated with the metrics
// in push gateway
func NewPushExporter(namespace, pushAddr, pushPort, jobName string) *PushExporter {
	var pe = &PushExporter{
		Meter:     view.NewMeter(),
		namespace: namespace,
		pushAddr:  pushAddr,
		pushPort:  pushPort,
		jobName:   jobName,
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
func (pe *PushExporter) Record(ctx context.Context, measurement interface{}, attachments map[string]interface{}) {
	pe.Meter.Record(tag.FromContext(ctx), measurement, attachments)
}

// PushMetrics extracts all of the data from the views and pushes them to the pushgateway
// Errors are being returned on a channel in case we encounter multiple
func (pe *PushExporter) PushMetrics() {
	client := &http.Client{}
	// push metrics for each view registered to the Meter
	for _, view := range pe.views {
		rows, err := pe.Meter.RetrieveData(view.Name)
		if err != nil {
			continue
		}
		metricName := fmt.Sprint(pe.namespace, "_", view.Name)
		helpString := fmt.Sprint("#HELP ", metricName, " ", view.Description, "\n")
		typeString := fmt.Sprint("#TYPE ", metricName, " ", getType(view.Aggregation.Type), "\n")
		for _, row := range rows {
			reqData := fmt.Sprint(helpString, typeString, metricName, formatRowData(row, view), "\n")
			req, err := http.NewRequest(http.MethodPost, pe.buildURLString(), bytes.NewBuffer([]byte(reqData)))
			if err != nil {
				continue
			}
			req.Header.Set("Content-Type", "plain/text; charset=utf-8")
			_, err = client.Do(req)
			if err != nil {
				continue
			}
		}

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

func formatRowData(row *view.Row, v *view.View) string {
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
		formattedData = fmt.Sprint(formattedData, "} ", row.Data.(*view.DistributionData).Count)
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
	default:
		returnType = "untyped"
	}
	return returnType
}

// The below was used for testing purposes without actually writing tests yet.
// Note that the package name must change to main and you will need a local
// instance of pushgateway running for the test to work.

// func main() {
// 	exporter := NewPushExporter("demo", "http://localhost", ":9091", "myJob")
// 	exporter.SetInstance("thebest")
// 	exporter.RegisterViews(loopCountView)
// 	for i := int64(0); i < 5; i++ {
// 		ctx, _ := tag.New(context.Background(), tag.Upsert(keyMethod, "main"))
// 		exporter.Record(ctx, []stats.Measurement{mLoops.M(1)}, nil)
// 	}
// 	exporter.PushMetrics()
// }

// // The measure and view to be used for demo purposes
// var keyMethod, _ = tag.NewKey("method")
// var mLoops = stats.Int64("demo", "The number of loop iterations", "1")
// var loopCountView = &view.View{
// 	Measure: mLoops, Name: "demo",
// 	Description: "Number of loop iterations",
// 	Aggregation: view.Count(),
// 	TagKeys:     []tag.Key{keyMethod},
// }
