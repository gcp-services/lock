# Lock
Lock is a cloud native distributed service for issuing and maintaining fine grained locks. Lock is meant to be used in systems where requests number in the hundreds of thousands to millions of requests per second, however may be used at any scale.

## Backends

Lock has modular support for multiple backends for storing locks in various durable and in memory stores.

## Why?

Distributed locks can be messy to implement, and depending on the mechanism for storing locks, implementations can vary wildly between languages and stacks. Lock offers a unified interface for lock management, exposed via gRPC, in order to make lock implementation homogeneous between all services in your stack.