# Raft implementation of a Key-Value Store

This project implements a key-value store using the Raft consensus algorithm. It provides a fault-tolerant, distributed system for storing and retrieving data.

## AVAILABLE COMMANDS

This Raft implementation supports the following distributed data structure commands:

## Basic Key-Value Operations

- `set <key> <value>` - Store a key-value pair
- `get <key>` - Retrieve the value for a key
- `rm <key>` - Remove a key-value pair
- `incr <key>` - Increment a numeric value
- `decr <key>` - Decrement a numeric value
- `keys` - Get all keys in the storage

## List Operations

- `lpush <key> <element> [element2] [element3]` - Prepend elements to a list
- `lpop <key>` - Remove and return the leftmost element from a list
- `lindex <key> <index>` - Get the element at the specified index in a list
- `llen <key>` - Get the length of a list

## Set Operations

- `sadd <key> <member> [member2] [member3]` - Add one or more members to a set
- `srem <key> <member> [member2] [member3]` - Remove one or more members from a set
- `sismember <key> <member>` - Test if member is in the set (returns 1 if true, 0 if false)
- `sinter <key1> [key2] [key3]` - Return the intersection of multiple sets
- `scard <key>` - Get the cardinality (number of elements) of a set

## Hash Operations

Hash operations allow you to store field-value pairs within a single key, similar to Redis hashes.

- `hset <key> <field> <value> [field value ...]` - Set field-value pairs in a hash
- `hget <key> <field>` - Get the value of a hash field
- `hmget <key> <field> [field2] [field3]` - Get the values of multiple hash fields
- `hincrby <key> <field> <increment>` - Increment the integer value of a hash field by the given number

## Cluster Management

- `status` - Get the current cluster status

All commands are replicated across the Raft cluster for consistency and durability.

## RESOURCES

- [Raft Consensus Algorithm Paper](https://raft.github.io/raft.pdf)
- [Distributed Systems 6.2: Raft](https://www.youtube.com/watch?v=uXEYuDwm7e4)

## TODO

- [ ] Implement Snapshotting
