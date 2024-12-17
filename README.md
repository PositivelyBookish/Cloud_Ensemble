# Distributed Image Classification System

This project implements a **Distributed Image Classification System** for agricultural purposes. It uses multiple **machine learning models** (AlexNet, MobileNet, ConvNeXt-Tiny) for image classification of plant health, providing predictions asynchronously. The system includes a **Go-based backend** with RabbitMQ for message queuing, a Python-based image classification script, and NGINX for load balancing.

---

## Prerequisites

Before running the project, ensure you have the following tools installed:

- **Go** (>= v1.16) — [Download Go](https://golang.org/doc/install)
- **Python** (>= 3.8) — [Download Python](https://www.python.org/downloads/)
- **RabbitMQ** — [Install RabbitMQ](https://www.rabbitmq.com/download.html)
- **PostgreSQL** — [Install PostgreSQL](https://www.postgresql.org/download/)

---

## Setup Instructions

### Step 1: Clone the Repository
Clone the project to your local machine:
```bash
git clone https://github.com/hjani-2003/Cloud_Computing_Project.git
cd Cloud_Computing_Project
```

---

### Step 2: Install Python Dependencies
Create a virtual environment and install the required Python libraries:
```bash
# Create a Python virtual environment
python -m venv venv

# Activate the virtual environment
source venv/bin/activate   # For Linux/Mac
venv\Scripts\activate      # For Windows

# Install the required Python packages
pip install -r requirements.txt
```

---

### Step 3: Run the Consumers
The consumers handle image classification tasks for different models and database insertion. Run the following commands **in separate terminals** to start all consumers:

```bash
# Run AlexNet consumer
go run alexnet-consumer.go

# Run MobileNet consumer
go run mobilenet-consumer.go

# Run ConvNeXt-Tiny consumer
go run convnext_tiny-consumer.go

# Run Database insertion consumer
go run database-consumer.go
```

---

### Step 4: Start the Go Server
Once the consumers are running, start the Go backend server:
```bash
go run server/server.go
```

The server will listen on `http://localhost:8081` for HTTP requests.

---

### Step 5: Access the Application
- Upload images to the classification endpoint: `http://localhost:8081/classify`
- The Go server will process the image, send tasks to RabbitMQ, and forward them to appropriate consumers.
- Results will be inserted into PostgreSQL and returned as a JSON response.

---

## Workflow Overview

1. **NGINX**: Acts as a **reverse proxy** to forward client requests to the Go backend.
2. **Go Backend**: Handles HTTP requests, interacts with RabbitMQ, and sends image classification tasks.
3. **RabbitMQ**: Queues classification requests and ensures **asynchronous processing** by different consumers.
4. **Consumers**:
    - Written in Go and Python.
    - Process classification tasks using pre-trained ML models (AlexNet, MobileNet, ConvNeXt-Tiny).
    - Insert classification results into the PostgreSQL database.
5. **Database**: Stores metadata and results for each classification task.

---

## Notes

- **Timeouts**: NGINX timeouts are configured for large image uploads and slow network connections.
- **Message Queue**: RabbitMQ ensures efficient task distribution and scalability.
- **Load Balancing**: NGINX enables horizontal scaling for the Go backend server.

---

## Troubleshooting

- Ensure RabbitMQ and PostgreSQL services are running.
- If Python dependencies fail to install, verify `pip` version with `pip --version`.
- For debugging, check RabbitMQ queues via the management console (`http://localhost:15672`).

---

## Future Enhancements

- Add support for image compression and optimized transmission.
- Implement advanced fault-tolerance for RabbitMQ consumers.
- Integrate additional deep-learning models.

---
