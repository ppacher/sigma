# Quick Start

This guide will get you started using Sigma. We will download the prebuilt release binary and use Docker for scheduling function nodes. The quick-start guide will setup the `sigma/nodejs` docker container for executing JavaScript based functions. 

## Installation

First we need to setup a working directory for our Sigma server:

```bash
mkdir ~/sigma
cd ~/sigma
```

Download the latest release binary from Github's release page [here]() und unpack it within the working directory:

```bash
wget https://github.com/homebot/sigma/releases/<release-name>.tar.gz
tar xfz ./<release-name>.tar.gz
chmod +x ./sigma
```

Next, we need to create the sigma configuration file `sigma.yaml` by running the following command:

```bash
./sigma server init --launcher=docker --add-type=js
```

Now that we have Sigma installed and configured, it's time to fire-up the server:

```bash
./sigma server --log-events
```

## Your first Function

A function specification always resides in a separate YAML file. Since most functions will have additional files it's a good practice to setup
a sub-directory for each function. Fortunately, the `sigma` command line client can scaffold new functions:

```bash
./sigma new --name=myFunction --type=js
```

This will create a directory and a basic function definition. Since we also specified `--type=js` the client will also construct a basic JavaScript function.

```bash
myFunction
   |- myFunction.yaml
   |- handler.js
```

`myFunction.yaml`
```yaml
name: myFunction
version: 0.1
content:
    file: handler.js
```

`handler.js`
```javascript
import * as iotc from 'homebot-sdk';

export.handler = function(topic, event) {
    //
    // Your code goes here
    //

    return `Nice to know about "${topic}"`;
}
```

Now that we have a basic function it's time to submit it to Sigma:

```bash
$ ./sigma submit ./myFunction
Function submitted to sigma.
Function-URN: urn::sigma::function:myFunction
```

```bash
$ ./sigma list
myFunction:
    type: JS
```

## Executing a function

In addition to configuring function triggers one can always request the execution of a function manually:

```plain
$ ./sigma exec --name myFunction --type "My first Function" --payload "is pretty cool"
Executing function urn::sigma::function:myFunction
Selected Node: urn::sigma::function:myFunction/<random-instance-id>

Nice to know about "My first Function"
```

## Inspecting a function

In order to inspect the state of a function `sigma inspect` can be useful:

```plain
$ ./sigma inspect --name myFunction
Name: myFunction
URN: urn::sigma::function:myFunction
Last-Invocation: Tue, 6 2017 11:31 AM
Nodes:
  <random-instance-id>  ACTIVE 

Events:
+ urn::sigma::function::myFunction deploying function instance
+ urn::sigma::function::myFunction/<random-instance-id> executed in 10ms
```