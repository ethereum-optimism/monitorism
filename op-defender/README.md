# Defenders

_op-defender_ is an active security service allowing to provide automated defense for the OP Stack.

The following the commands are currently available:

```bash
NAME:
   Defender - OP Stack Automated Defense

USAGE:
   Defender [global options] command [command options]

VERSION:
   0.1.0-unstable

DESCRIPTION:
   OP Stack Automated Defense

COMMANDS:
   psp_executor  Service to execute PSPs through API.
   version       Show version
   help, h       Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h     show help
   --version, -v  print the version
```

Each _defenders_ has some common configuration, that are configurable both via CLI or environment variables with defaults values.

### PSP Executor Service

<img width="3357" alt="image" src="https://github.com/user-attachments/assets/4a63119d-b5e1-4b86-b80e-38d21366ef0b">

The PSP Executor Service is made for executing PSP onchain faster, to increase our readiness and speed in our response. 

| `op-defender/psp_executor` | [README](https://github.com/ethereum-optimism/monitorism/blob/main/op-defender/psp_executor/README.md) |
| -------------------------- | ------------------------------------------------------------------------------------------------------ |
