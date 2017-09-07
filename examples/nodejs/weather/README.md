# Weather example

This example provides a function to query current weather conditions using [weather-js](https://www.npmjs.com/package/weather-js). It is based on [sigma-bootstrap-typescript](https://github.com/homebot/sigma-bootstrap-typescript).

To build and use the example, execute the following commands:

```bash
# Install dependencies and required build tools
npm install

# Build the final function bundle to dist/index.bundle.js
npm run build

# Deploy the function on Sigma
sigma deploy ./weather.yaml

# Test it
sigma exec --name weather --payload 'Vienna, AT'
```