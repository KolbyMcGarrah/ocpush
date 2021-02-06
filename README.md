# ocpush

## Overview
ocpush is a metrics exporter for pushing opencensus metrics to prometheus pushgateway. 

## Usage
In progress

## Helpful resources
Prometheus api data format: https://docs.google.com/document/d/1ZjyKiKxZV83VI9ZKAXRGKaUKK2BIWCT7oiGBKDBpjEY/edit#

Prometheus pushgateway repo: https://github.com/prometheus/pushgateway
command for local pushgateway: 
``` 
docker pull prom/pushgateway
docker run -d -p 9091:9091 prom/pushgateway 
```