# Messaging System Assignment

This project implements a distributed message streaming system in Go, featuring a reader and a writer communicating via Redis Streams. It includes Kubernetes deployment, observability with Prometheus and graceful handling of Redis stream processing.

---

## Architecture

```
+-----------+          Redis Stream         +-----------+
 |  Reader |   --->   [line_stream]  --->    |  Writer |
+-----------+                               +-----------+
      |                                           |
/data/input.txt                           /data/output.txt
```


```
[input.txt] --> Reader --> Redis Stream --> Writer --> [output.txt]
```

- **Reader** reads lines from a text file and sends them to a Redis Stream.
- **Writer** reads from the stream and writes each line to a new file.
- Communication between **Reader** and **Writer** is done over **RESP**.

---

## Components

### Reader
- Watches a file (`input.txt`)
- Streams new lines into a Redis stream (`line_stream`)
- Metrics exposed on `/metrics` at port `8080`
- Health check available at `/healthz`

### Writer
- Uses Redis `XREADGROUP` to consume lines
- Acknowledges messages via `XACK`
- Writes lines to `output.txt`
- Metrics exposed on `/metrics` at port `8080`
- Health check available at `/healthz`

---

## Kubernetes Deployment

*Tested on Kubernetes v1.32.2*

Includes manifests for:
- Producer and consumer deployments + services
- Prometheus and Prometheus Adapter
- Autoscaling based on stream backlog
- PodDisruptionBudget for consumer

### Getting started 

Start by deploying core components 
```
kubectl apply -f ./kubernetes/prometheus-deployment.yaml 
kubectl apply -f ./kubernetes/prometheus-configmap.yaml 
kubectl apply -f ./kubernetes/redis.yaml
```
Deploy reader and verify data is being generated and sent to Redis

```
kubectl apply -f ./kubernetes/reader.yaml
kubectl logs -l app.kubernetes.io/name=reader -f
```

Deploy writer and verify data is being written to output.txt
```
kubectl apply -f ./kubernetes/writer.yaml
kubectl logs -l app.kubernetes.io/name=writer -c tail -f
```


---

## Health Checks

Both services expose:
- `/healthz` for Kubernetes probes
- `/metrics` for Prometheus scraping

---

## Metrics

Prometheus metrics include:

| Metric Name                          | Description                             |
|-------------------------------------|------------------------------------------|
| `lines_read_total`                  | Total lines added to Redis stream        |
| `lines_written_total`               | Total lines processed by writer.         |

---