# RESOURCES

- [Raft Consensus Algorithm Paper](https://raft.github.io/raft.pdf)
- [Distributed Systems 6.2: Raft](https://www.youtube.com/watch?v=uXEYuDwm7e4)

# AVAILABLE COMMANDS

This Raft implementation supports the following distributed data structure commands:

## Basic Key-Value Operations

- `set <key> <value>` - Store a key-value pair
- `get <key>` - Retrieve the value for a key
- `rm <key>` - Remove a key-value pair
- `incr <key>` - Increment a numeric value
- `decr <key>` - Decrement a numeric value

## List Operations

- `lpush <key> <element> [element2] [element3]` - Prepend elements to a list
- `lpop <key>` - Remove and return the leftmost element from a list
- `lindex <key> <index>` - Get the element at the specified index in a list

## Cluster Management

- `status` - Get the current cluster status

All commands are replicated across the Raft cluster for consistency and durability.

# TODO

- [ ] Implement Snapshotting
