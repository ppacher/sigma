# Sigma `Σ`

Sigma is a extensible and hackable Function-as-a-Service (Faas) platform written in Golang.

> `Sigma` is currently under active development and a first stable version should be released soon. Stay tuned ;)

See [homebot.github.io/sigma](https://homebot.github.io/sigma) for in-progress documentation.

## Roadmap

**v0.3**

- [ ] Function versioning
- [ ] Rolling updates
- [ ] Live-reload configuration
- [ ] OpenTracing
- [ ] Event dispatcher for MQTT
- [ ] Event dispatcher for AMQP

**v0.2**

- [ ] Loading functions from a storage backend
- [ ] Trigger plugins based on [hashicorp/go-plugin](https://github.com/hashicorp/go-plugin)
- [ ] Prometheus metrics
- [ ] Support to submit archives as functions


**v0.1** *expected 2017-09*

- [ ] Test cases
- [ ] Basic load balancing (round-robin)
- [ ] Simple command-line client
- [X] Launcher: Docker
- [X] Launcher: Process (launching functions as native processes)
- [X] Support for auto-scaling policies
- [X] Simple, generic metric interface
- [X] Plugable trigger support (compile time)
- [X] Trigger conditions
- [X] function variables/function templates
- [X] gRPC server interface