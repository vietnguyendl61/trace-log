# Task Distribution System - Improved Design

## 1. System Overview

The system consists of **2 main services**:

- **Task Manager Service** (Manager)
- **Task Worker Service** (Worker) – supports horizontal scaling with multiple instances

**Objectives:**
- Manage tasks organized by **Group** (each group has a unique `group_id`).
- Support group lifecycle events: `processing`, `pause`, `resume`, `done`.
- Distribute tasks **fairly** across multiple active groups.
- Support long-running tasks (30 minutes to several hours) without crashing the message queue.
- Provide near real-time task completion percentage.
- Ensure high reliability when scaling workers.

## 2. Overall Architecture
```mermaid
graph TD
    A[External API] -->|Event: processing/pause/resume/done| B[Task Manager Service]
    B <--> C[(MongoDB)]
    
    B -->|Scheduler<br/>Batch 20-50 task<br/>Weighted Fair| D[Task Queue<br/>RabbitMQ]
    
    D --> E[Worker Instance 1..N]
    E -->|Ack ngay| D
    E -->|Push task| F[Background Processor<br/>(thread pool / Celery)]
    F <--> C
    
    subgraph "Real-time Monitoring"
    B -->|Query % done mỗi 5s| G[Dashboard / API]
    end
```

### 3. Service Details

#### 3.1 Task Manager Service

**Main Responsibilities:**

- Acts as the single source of truth for task and group states.
- Manages the full lifecycle of each Group based on events.
- Distributes tasks using a fair distribution mechanism across multiple groups.
- Monitors and reclaims hanging tasks.

**Key Components:**

- **Group Status**: `pending` → `processing` → `paused` → `resume` → `done`
- **Task Status**: `pending` → `in_progress` → `completed` → `failed`

**Distribution Scheduler** (runs every 5–10 seconds):
- Scans all groups with status `processing`.
- Calculates current % completed for each group.
- Pushes tasks in small batches (default 20–50 tasks per batch).
- Uses **Weighted Round-Robin** or **Proportional Fair Share** to ensure fair allocation between active groups.
- Prioritizes groups with lower completion percentage to maintain balance.

**Event Handling:**

- `processing`: Marks group as processing and enables the scheduler to start pushing tasks.
- `pause`: Marks group as paused and stops pushing new batches for that group.
- `resume`: Marks group as processing and resumes pushing tasks.
- `done`: Marks group as done, stops all distribution, and optionally triggers cleanup.

**Reclaim Mechanism:**

- Periodically scans tasks in `in_progress` state where `claim_timestamp` has expired (e.g., older than 2 hours).
- Changes status back to `pending` so the scheduler can redistribute them (with `retry_count` tracking to prevent infinite loops).

#### 3.2 Task Worker Service

**Main Responsibilities:**

- Supports easy horizontal scaling (multiple instances consuming from the queue simultaneously).
- Receives tasks from the queue, processes them, and updates results.
- Handles very long-running tasks without breaking RabbitMQ connections.

**Worker Design (Critical Pattern):**  
Each worker instance follows the **Early Acknowledgment + Background Processing** pattern:

1. Receives message from RabbitMQ with `prefetch_count = 1`.
2. Acknowledges immediately (manual ack) to prevent connection/channel timeout.
3. Pushes the task into a Background Processor (internal queue, Celery, RQ, or ThreadPoolExecutor).
4. The background processor performs the actual heavy work (can run 30 minutes to several hours).
5. Updates progress and final result into MongoDB.
6. Optionally sends progress heartbeat every 30–60 seconds for very long tasks.

**Recommended RabbitMQ Configuration:**

- `prefetch_count = 1` (each worker handles only one task at a time).
- Manual acknowledgment.
- Use **Dead Letter Queue (DLQ)** for tasks that fail repeatedly.

---

### 4. Main Flow

1. External system sends event → Manager updates group status.
2. Distribution Scheduler runs periodically → pushes small batches of tasks to RabbitMQ (only for `processing` groups).
3. Worker receives task → acks immediately → forwards to Background Processor.
4. Background Processor executes the task → updates MongoDB (`in_progress` → `completed`).
5. Manager calculates completion percentage and exposes data for monitoring/dashboard.