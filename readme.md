Temporal Workflow SLO & Root Cause Correlator

**WeMakeDevs x SigNoz Hackathon — Track 2: Signals & Dashboards**

This repository contains a production-grade observability system designed for Temporal workflows, heavily utilizing advanced OpenTelemetry instrumentation and SigNoz Query Builder mastery. It goes beyond basic server metrics to correlate traces, metrics, and structured logs into a single pane of glass, enabling site reliability engineers (SREs) and AI Agents to automatically identify the root cause of complex distributed system failures.

## Prerequisites
### Phase 1: AWS Infrastructure Provisioning

1. Launch an EC2 Instance:
- AMI: Ubuntu Server 22.04 LTS (HVM)
- Instance Type: t3.large (2 vCPUs, 8 GiB Memory — required to comfortably run Temporal + the load generator).
- Storage: 20 GB gp3 root volume.

--- 

2. Configure the Security Group:
To ensure the telemetry pipeline and UI are secure, configure the inbound rules to only allow your specific IP address (My IP): open port
- Port 22 (TCP): For SSH access to the server.
- Port 8080 (TCP): To access the Temporal Web UI.
- Port 8000 (TCP): For application service routing/API access.
- Port 8088 (TCP): For additional metric endpoints or load generator UI.
- (Ensure all outbound traffic is allowed so the OTel Collector can reach your SigNoz instance).
