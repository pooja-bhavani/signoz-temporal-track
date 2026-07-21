Temporal Workflow SLO & Root Cause Correlator

**WeMakeDevs x SigNoz Hackathon — Track 2: Signals & Dashboards**

This repository contains a production-grade observability system designed for Temporal workflows, heavily utilizing advanced OpenTelemetry instrumentation and SigNoz Query Builder mastery. It goes beyond basic server metrics to correlate traces, metrics, and structured logs into a single pane of glass, enabling site reliability engineers (SREs) and AI Agents to automatically identify the root cause of complex distributed system failures.

## Prerequisites
### Phase 1: AWS Infrastructure Provisioning

1. Launch an EC2 Instance:
- AMI: Ubuntu Server 22.04 LTS (HVM)
- Instance Type: t3.large (2 vCPUs, 8 GiB Memory — required to comfortably run Temporal + the load generator).
- Storage: 20 GB gp3 root volume.
