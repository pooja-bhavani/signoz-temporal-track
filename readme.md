# Temporal Workflow SLO & Root Cause Correlator

**WeMakeDevs x SigNoz Hackathon — Track 2: Signals & Dashboards**

This repository contains a production-grade observability system designed for Temporal workflows, heavily utilizing advanced OpenTelemetry instrumentation and SigNoz Query Builder mastery. It goes beyond basic server metrics to correlate traces, metrics, and structured logs into a single pane of glass, enabling site reliability engineers (SREs) and AI Agents to automatically identify the root cause of complex distributed system failures.

## Prerequisites

### Phase 1: AWS Infrastructure Provisioning

#### 1. Launch an EC2 Instance:
- AMI: Ubuntu Server 22.04 LTS (HVM)
- Instance Type: t3.large (2 vCPUs, 8 GiB Memory — required to comfortably run Temporal + the load generator).
- Storage: 20 GB gp3 root volume.


#### 2. Configure the Security Group:
   
To ensure the telemetry pipeline and UI are secure, configure the inbound rules to only allow your specific IP address (My IP): open port
- Port 22 (TCP): For SSH access to the server.
- Port 8080 (TCP): To access the Temporal Web UI.
- Port 8000 (TCP): For application service routing/API access.
- Port 8088 (TCP): For additional metric endpoints or load generator UI.
- Port 4317 (TCP): For (OTLP gRPC)
- Port 4318 (TCP): For (OTLP HTTP)
- (Ensure all outbound traffic is allowed so the OTel Collector can reach your SigNoz instance).

<img width="1470" height="837" alt="Screenshot 2026-07-21 at 11 37 47 PM" src="https://github.com/user-attachments/assets/bcdc296d-91c5-4a65-9c25-78b29f44db77" />

### Phase 2: Server Preparation (SSH & Docker)
Once the instance is running, SSH into your t3.large and install the required dependencies:

#### 1. SSH into your Ubuntu 22.04 instance
```
ssh -i your-key.pem ubuntu@<your-ec2-public-ip>
```

#### 2. Update packages and install Docker & Git
```
sudo apt update && sudo apt upgrade -y
sudo apt install -y git docker.io docker-compose-v2
```

#### 3. Add the ubuntu user to the docker group ( this avoids need to use sudo for every docker commands)

```
sudo usermod -aG docker ubuntu
newgrp docker
```

### Phase 3: Application Deployment

Now, clone the hackathon repository and configure the environment to connect to SigNoz.

#### 1. Clone the repository

```
git clone https://github.com/pooja-bhavani/signoz-temporal-track.git
cd signoz-temporal-track
```
#### 2. Configure Environment Variables

Replace the IP below with your actual SigNoz OTLP HTTP endpoint (Port 4318)

```
echo "SIGNOZ_ENDPOINT=http://<your-signoz-ip>:4318" > .env
echo "RPS=3" >> .env
echo "TEMPORAL_ADDRESS=temporal-server:7233" >> .env
```

#### 3. Build the Go binaries and boot the cluster

```
docker compose up --build -d
```

<img width="1459" height="857" alt="image" src="https://github.com/user-attachments/assets/f6055f2e-6ef4-43c6-ba1c-18bb9ad60d03" />

### Phase 4: Verification & Observability

Verify that the services are running and that OpenTelemetry data is successfully flowing out of your EC2 instance into SigNoz.

#### 1. Check that all 7 containers are "Up"

```
docker compose ps
```
#### 2. Verify OpenTelemetry Collector is exporting traces/metrics to SigNoz without errors

```
docker compose logs --tail=10 otel-collector
```

#### 3. Verify the Temporal Worker is processing the load
```
docker compose logs --tail=10 worker
```

<img width="1470" height="810" alt="Screenshot 2026-07-21 at 11 46 14 PM" src="https://github.com/user-attachments/assets/c8d16037-c863-4a20-b6b7-450acb473a7b" />


#### Check UI

Open your web browser and check the SigNoz Because of your Security Group rules, only you will be able to see the live Temporal UI executing the 6,000+ workflows.

```
http://<your-ec2-public-ip>:8080
```

<img width="1468" height="885" alt="Screenshot 2026-07-21 at 11 49 15 PM" src="https://github.com/user-attachments/assets/52e0586a-519d-42cd-af7f-ea1332728c50" />

#### Check the temporal UI
```
http://<your-ec2-public-ip>:8088
```
<img width="1469" height="883" alt="image" src="https://github.com/user-attachments/assets/35a255ff-2c6f-471a-a2b0-1e5ac161fe2b" />

<img width="1464" height="882" alt="image" src="https://github.com/user-attachments/assets/195c2b84-ad75-4adf-b514-3035cc4e779b" />



















