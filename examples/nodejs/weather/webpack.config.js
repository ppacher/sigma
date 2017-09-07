var webpack = require('webpack'),
path = require('path');
module.exports = {
    context: __dirname + '/src',
    target: "node",
    entry: {
        index: './main.ts'
    },
    output: {
        path: __dirname + '/dist',
        publicPath: '/',
        filename: '[name].bundle.js',
        libraryTarget: "commonjs-module",
    },
    resolve: {
        extensions: ['.ts', '.js']
    },
    module: {
        loaders: [
            { test: /\.ts$/, loaders: ['ts-loader'], exclude: /node_modules/ }
        ]
    }
}