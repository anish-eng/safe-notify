# 🚀 Safe-Notify  
### A Fault-Tolerant, Event-Driven Notification Pipeline

Safe-Notify is a **production-inspired asynchronous notification delivery system** designed to reliably deliver notifications in the presence of failures, retries, spikes, and unreliable third-party services.

This project focuses on addressing a **real distributed systems problems** that occur in production environments:  
retry storms, duplicate sends, worker and process crashes, delayed retries, and operational recovery in case of delivery failure.

---

# Tech Stack- 

- React.js for UI and observability dashboard
- Golang for the backend API endpoints and asynchronous workers and retry scheduler
- Kafka for storing the task ids as an intermediate state
- AWS-SES- For the email sending service
- AWS DynamoDB for storing the task information

## 🧠 The Problem This System Solves

Sending notifications is difficult at scale when there are thousands of notifications to be sent and each component of the system may not always work reliably. 


In real systems, notification delivery fails for many reasons:

- Email/SMS providers fail intermittently
- Network requests time out
- Workers crash mid-processing
- Traffic spikes overwhelm and slow down synchronous systems
- Retrying too aggressively causes duplicate notifications sent
- Retrying too slowly delays critical alerts
- Database polling does not scale and slows down the application severely
  

A synchronous design like:

HTTP request → send email → return response


fails silently, does not scale, and is unsafe because incase of a sending failure we don't know the point of failure and cannot ensure reliable message delivery because we dont know if the notification was successfully delivered or not. Trying to redeliver a message may cause duplicate messages which is not ideal. Also there is no record stored of what happened and why



I developed Safe-Notify to help solve these problems.

---

## 🎯 Design Goals

Safe-Notify is built around the following principles:

- Asynchronous by default -Since the API gateway interacting with the user and the actual task sending mechanism is separated, these processes do not have to occur sequentially.  
- Explicit failure handling and preventing blocking of tasks and slowdown of system  
- Controlled retries of operations with an exponential backoff to avoid successive consecutive retries and overloading the system
- Idempotent processing - ensuring notifications are not sent twice for the same task by storing a unique idempotency key
- Durable state- Real time status of each notification is stored in a permanent database, so there is a record of the status of each notification and nothing is lost
- Observability & replay - Through displaying the real-time status of a message sent through a react.js dashboard and allowing retry to send notifications when maximum limit of trial is reached
- Horizontally scalable - The prototype is built assuming a single worker that is processing the incoming tasks, but is designed such that more workers/ processing resources can be added to ensure quick and efficient processing
-Simlulating system failures and chaos testing to understand how the system would behave incase of failures.
---

## 🏗 High-Level Architecture



Frontend (React)
↓
REST API (Go)
↓
DynamoDB (Task State)
↓
Kafka (Main Queue)
↓
Worker Service
↓
AWS SES (Email Provider)
↓
Kafka Retry Queue
↓
Retry Scheduler
↓
Kafka Main Queue


Each component is **intentionally decoupled** so that failures are isolated and recoverable.

---

## 🔄 End-to-End Data Flow

### 1️⃣ Event Creation (Ingress)

- User submits an event from the UI
- API validates input
- A task record is created in DynamoDB with:
  - `status = PENDING`
  - `attempt_count = 0`
- Task ID is published to Kafka (`safe-notify-tasks`)

**Why this matters:**  
The API returns immediately and never blocks on delivery. The API is not tasked with delivering the message so it will not get blocked with that task.

---

### 2️⃣ Task Dispatch (Kafka Main Queue)

Kafka acts as a **durable buffer** between ingestion and processing.

**Kafka provides:**
- Load handling
- Message durability
- Horizontal scaling
- Failure isolation

This prevents traffic spikes from overwhelming workers.

---

### 3️⃣ Worker Processing

Workers:
- Consume task IDs from Kafka
- Fetch task state from DynamoDB
- Atomically **claim the task**
- Attempt delivery via AWS SES

The claim step ensures:
- Only one worker processes a task
- Duplicate Kafka deliveries do not cause duplicate sends because a single worker is processing it at any point.

---

### 4️⃣ Success Path

If delivery succeeds:
- Task status is updated in DynamoDB → `SENT`
- Attempt count incremented
- Kafka offset committed

This ensures **exactly-once effects**, even with at-least-once delivery.

---

### 5️⃣ Failure Path (Controlled Retry)

If delivery fails:
- Error is recorded
- Attempt count incremented
- Retry is scheduled with backoff depending on which attempt it is.
- Task is published to **retry queue**

Backoff example:


Attempt 1 → retry in 2s
Attempt 2 → retry in 5s
Attempt 3 → DLQ


---

### 6️⃣ Retry Scheduler (Delay Holder)

Kafka does **not support delayed messages**.

Instead, such messages are published to a  **Retry Scheduler service** that:

- Consumes retry messages
- Sleeps until `next_retry_at`
- Re-publishes the task to the main queue


The **scheduler holds the delay** instead of kafka.

This avoids:
- Blocking workers
- Busy-waiting of the workers
- Unnecessary Database polling 

---

### 7️⃣ Dead Letter Queue (DLQ)

When `max_attempts` is exceeded:
- Task is marked `DLQ`
- It is no longer retried automatically
- Task remains visible and replayable

This prevents infinite retry loops if there is something really wrong with the system.

---

### 8️⃣ Manual Replay

Manual intevention can be done incase a task has failed to send after 3 successive attempts- Users can:
- Inspect failed tasks
- Resend the tasks from the UI and repeat the notification sending process as if they are working with a new task.
- Safely re-enqueue tasks to Kafka without duplication



---

## 🧩 Core Components

### 🌐 Frontend (React + Vite)

**Purpose**
- Create notification events
- Visualize task lifecycle
- Resend failed tasks back to kafka for another sending attempt
---

### ⚙ API Service (Go)

**Purpose**
- Facilitates interaction between the UI and DynamoDB
- Publishing incoming events to kafka for further processing


---

### 🗄 DynamoDB (State Store)

**Purpose**
- Single source of truth for task status and information which the UI and worker access and update.
- Preventing duplication notifications from being sent(idempotency) and reliable storing of task information

**Why DynamoDB?**
- Fast conditional writes
- Strong database consistency options
- Scales well, independent of number of workers

---

### 📬 Kafka – Main Queue

**Purpose**
- Decouple ingestion from processing
- Prevents and manages load spikes from happening
- Enable horizontal scaling(more kafka queues can be added for parallel processing of more messages)

**Why Kafka over DB polling?**
- Polling databases constantly for status change is not scalable because of excessive reads required.
- Kafka is event-driven and notifies the worker immediately once it has a task to deliver
- Reduces database load and interacts with the worker directly.

---

### 🔁 Kafka – Retry Queue

**Purpose**
- Stores retry intent
- Separate retry traffic from new traffic
- Keep workers fast(they do not have to be blocked because of retry messages)

**Why a separate topic?**
- Clear separation of responsibilities and reduced load on the worker
- Easier observability of task status
- Prevents retry storms when several entries are retried together

---

### ⏱ Retry Scheduler

**Purpose**
- Centralized delay handling 
- Backoff enforcement

**Why not sleep inside workers?**
- Workers should not be blocked because of retrying tasks 
- Adding a separate scheduler process scales independently and helps manage retry tasks efficiently

---

### 🧑‍🏭 Worker Service

**Purpose**
- Consumes task id's from Kafka and queries pending tasks from DynamoDB.
- Then it claims a task and then attempts to send notifications via AWS SES mailing service.
- It adds failed tasks to the retry queue in Kafka.


**Key guarantees**
- Idempotent execution
- Safe retries
- Crash recovery(If a worker fails while processing a task, another worker can take up that task)

---

### 📧 AWS SES

**Purpose**
- Real email notification delivery service that sends the final notification
- Simulates third-party unreliability and behavior like in real distributed systems.


---


## 🔍 Observability & Operations

- Task status visible in UI
- Attempt counts tracked
- Last error recorded
- Retry timing explicit
- Logs across all services

This makes the system **easily debuggable**.

---

## 🧠 Why This Design Works

This system aims to be a scalable way to deliver notifications like in a real production system:

- Event-driven asynchronous processing
- Explicit retry task pipeline
- Storing durable state of tasks reliably.
- Clear understanding and management of failure
- Reliable and consistent notification delivery without delays and errors.


🔮 Future Improvements

- Multiple retry tiers (short/long delays)

- Adding filtering by task status, UX improvements

- Rate-limiting per recipient

- Add more notification providers

- Metrics with Prometheus/Grafana

- Partitioned Kafka topics

- Cloud deployment and infrastructure management (ECS / EKS)


---

## 🚀 Running the System Locally

```bash
docker compose up -d


Then run each service:

go run cmd/api/main.go
go run cmd/worker/main.go
go run cmd/scheduler/main.go


Frontend:

cd frontend
npm install
npm run dev

