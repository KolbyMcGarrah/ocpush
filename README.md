# ocpush

## Overview
ocpush is a metrics exporter for pushing opencensus metrics to prometheus pushgateway. It was designed to allow for opencensus metrics to be used within cron jobs without having to wait for an extended period of time after the job finishes. This allows for more consistent metrics between cron jobs, and services within an organization.

## Usage
ocpush creates its own Meter object which is necessary to be able to access view data. Because of this, all Views need to be registered to the Meter, and all Measures need to be recorded to the Meter which can be done using the provided methods from the PushExporter struct. Because of this, custom metrics are easily added to the PushExporter, but commonly used libraries that utilize opencensus metrics may not work well with it. Custom libraries may need to be made to integrate with PushWorker.

The basic functionality for using the push worker consists of the following actions:
* Create a new PushExporter object.
    * `NewPushExporter(isTest bool, namespace, pushAddr, pushPort, jobName string) *PushExporter`
* Register views to the PushExporter.
    * `func (pe *PushExporter) RegisterViews(views ...*view.View) error`
* Record Measures on the PushExporter.
    * `func (pe *PushExporter) Record(ctx context.Context, measurement []stats.Measurement, attachments map[string]interface{})`
* PushMetrics after job completes.
    *  `func (pe *PushExporter) PushMetrics()`

A PushExporter object has the following fields:
Field    | Data Type | Description
-------- |---------- |-------------
Meter    | view.Meter | Meter is the object collects data from the views to be able to push to the gateway
namespace | string | The namespace for the cron job metrics.
pushAddr | string | The address to the pushgateway
pushPort | string | The port that the pushgateway is accessible on
instance | string | The instance name of the job/service. Can be set using the `SetInstance` method
jobName | string | The name of the job
views | []*view.View | The list of views registered to the PushExporter
isTest | bool | Setting this to true will print metric data rather than push to the endpoint



## Helpful resources
Prometheus api data format: https://docs.google.com/document/d/1ZjyKiKxZV83VI9ZKAXRGKaUKK2BIWCT7oiGBKDBpjEY/edit#

Prometheus pushgateway repo: https://github.com/prometheus/pushgateway

command for local pushgateway: 
``` 
docker pull prom/pushgateway
docker run -d -p 9091:9091 prom/pushgateway 
```