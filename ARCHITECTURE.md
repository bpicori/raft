# Raft Key-Value Store Architecture

## Table of Contents

1. [System Overview](#system-overview)
2. [Core Components](#core-components)
3. [Raft Algorithm Implementation](#raft-algorithm-implementation)
4. [Data Flow](#data-flow)
5. [Architecture Diagrams](#architecture-diagrams)
6. [Testing Strategy](#testing-strategy)

## System Overview

This project implements a distributed key-value store using the Raft consensus algorithm. The system provides fault-tolerant, strongly consistent data storage with support for multiple data types including basic key-value operations, lists, sets, and hash operations.

### Key Features

- **Distributed Consensus**: Raft algorithm for leader election and log replication
- **Rich Data Types**: Basic operations, lists, sets, hashes
- **Fault Tolerance**: Handles node failures and network partitions
- **Strong Consistency**: All operations replicated before commit
- **Event-Driven Architecture**: Channel-based component communication
- **Persistent Storage**: Protocol buffer-based state persistence

## Core Components

### 1. Raft Module (`pkgs/raft/`)

Core consensus engine implementing the Raft algorithm.

**Key Responsibilities:**

- Leader election and term management
- Log replication across cluster nodes
- State machine persistence and recovery

**Key Data Structures:**

```go
type Raft struct {
    // Persistent state
    currentTerm  int32           // Latest term server has seen
    votedFor     string          // Candidate voted for in current term
    LogEntry     []*dto.LogEntry // Log entries
    CommitLength int32           // Index of highest committed entry
    
    // Volatile state
    currentRole      consts.Role      // Current role (Follower/Candidate/Leader)
    currentLeader    string           // Current leader ID
    votesReceivedMap sync.Map         // Votes received in current election
    sentLength       map[string]int32 // Log entries sent to each follower
    ackedLength      map[string]int32 // Log entries acknowledged by each follower
}
```

### 2. Application Layer (`pkgs/application/`)

Implements the key-value store logic and command processing.

**Key Responsibilities:**

- Processing client commands
- Maintaining in-memory data store (`sync.Map`)
- Applying committed log entries to state machine

### 3. Event Management (`pkgs/events/`)

Central event-driven communication system using Go channels.

**Key Responsibilities:**

- Coordinating communication between components
- Managing timers for elections and heartbeats
- Handling asynchronous command processing

### 4. TCP Communication (`pkgs/tcp/`)

Network layer handling inter-node and client communication.

**Key Responsibilities:**

- TCP server for accepting connections
- Protocol buffer message serialization/deserialization
- Synchronous and asynchronous RPC handling

### 5. Storage Layer (`pkgs/storage/`)

Persistent storage for Raft state machine.

**Key Responsibilities:**

- Persisting Raft state (term, voted for, log entries, commit length)
- Loading state on startup for crash recovery
- File-based storage using Protocol Buffers

**Key Message Types:**

- **Raft Messages**: `VoteRequest`, `LogRequest`, `LogResponse`
- **Command Messages**: Request/response pairs for each operation
- **State Messages**: `StateMachineState`, `NodeStatusResponse`

## Raft Algorithm Implementation

### Leader Election

1. **Follower Timeout**: When election timer expires, follower becomes candidate
2. **Vote Request**: Candidate increments term and requests votes from all nodes
3. **Vote Granting**: Nodes vote for first candidate in each term with up-to-date log
4. **Majority Check**: Candidate becomes leader if it receives majority votes
5. **Heartbeats**: Leader sends periodic heartbeats to maintain authority

### Log Replication

1. **Client Request**: Leader receives command from client
2. **Log Append**: Leader appends entry to its log with current term
3. **Parallel Replication**: Leader sends `LogRequest` to all followers
4. **Consistency Check**: Followers verify log consistency using prefix matching
5. **Acknowledgment**: Followers respond with success/failure
6. **Commit**: Leader commits entry when majority acknowledges
7. **Application**: Committed entries are applied to state machine

### Safety Properties

- **Election Safety**: At most one leader per term
- **Leader Append-Only**: Leaders never overwrite or delete log entries
- **Log Matching**: If two logs contain entry with same index and term, logs are identical up to that index
- **Leader Completeness**: If entry is committed in a term, it will be present in leaders of higher terms
- **State Machine Safety**: If server applies entry at index, no other server applies different entry at same index

## Data Flow

### Normal Operation (Leader Active)

```text
Client Request → TCP Server → Event Manager → Application Layer → Raft Leader
                                                                      ↓
                                                              Log Replication
                                                                      ↓
                                                              Followers Ack
                                                                      ↓
                                                              Commit & Apply
                                                                      ↓
                                                              Response to Client
```

### Leader Election Flow

```text
Election Timeout → Candidate State → Vote Requests → Vote Collection → Leader State
                                                                            ↓
                                                                    Heartbeat Timer
                                                                            ↓
                                                                    Regular Heartbeats
```

### Failure Scenarios

**Leader Failure:**

1. Followers detect missing heartbeats
2. Election timeout triggers new election
3. New leader elected with majority votes
4. New leader replicates any uncommitted entries

**Follower Failure:**

1. Leader detects failed replication
2. Leader continues with remaining followers
3. Failed follower rejoins and catches up via log replication

**Network Partition:**

1. Minority partition cannot elect leader (no majority)
2. Majority partition continues normal operation
3. Partitions heal and minority nodes catch up

## Architecture Diagrams

```text
Client Command Flow:
┌────────┐   TCP    ┌─────────┐   Event   ┌─────────────┐   Command   ┌─────────┐
│ Client │ ────────►│   TCP   │ ─────────►│    Event    │ ───────────►│  App │
│        │          │ Server  │           │  Manager    │             │ Layer   │
└────────┘          └─────────┘           └─────────────┘             └─────────┘
                                                 │                           
                                                 ▼                           
                                          ┌─────────────┐             
                                          │    Raft     │
                                          │    Core     │             
                                          └─────────────┘             
                                                 │
                                                 ▼
                                          ┌─────────────────┐
                                          │ Log Replication │
                                          └─────────────────┘
```
