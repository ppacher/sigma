# Sigma `Σ`

Sigma is a extensible and hackable Function-as-a-Service (Faas) platform written in Golang.

> `Sigma` is currently under active development and a first stable version should be released soon. Stay tuned ;)

See [homebot.github.io/sigma](https://homebot.github.io/sigma) for documentation.

## Roadmap

**v0.3**

- [ ] Function versioning
- [ ] Rolling updates
- [ ] Live-reload configuration

**v0.2**

- [ ] Persistent storage for functions
- [ ] Trigger plugins based on [hashicorp/go-plugin](https://github.com/hashicorp/go-plugin)
- [ ] Event dispatcher for MQTT
- [ ] Event dispatcher for AMQP


**v0.1** *expected 2017-09*

- [ ] Test cases
- [ ] Basic load balancing (round-robin)
- [ ] Simple command-line client
- [X] Support for auto-scaling policies
- [X] Simple, generic metric interface
- [X] Plugable trigger support (compile time)
- [X] Trigger conditions
- [X] function variables/function templates
- [X] gRPC server interface