[![Go Report Card](https://goreportcard.com/badge/github.com/1602/witness)](https://goreportcard.com/report/github.com/1602/witness)
[![Build Status](https://travis-ci.org/1602/witness.svg?branch=main)](https://travis-ci.org/1602/witness)
[![Coverage Status](https://img.shields.io/coveralls/github/1602/witness.svg)](https://coveralls.io/github/1602/witness?branch=main)

## Witness

Enables debugging of http requests via UI. It is like chrome devtools for go backend.

## How it works

It uses `httptrace.WithClientTrace` to make a `http.RoundTripper` eavesdropping on http connection. This allows detailed analysis of various http request stages. All information gathered then pushed using EventStream to UI running in browser.

## Usage

The idea is to observe http client you want to debug by calling `witness.DebugClient` with the http client in question. For example, if you "own" a client to make http calls to some API:

```
// init http client with any configuration you need
cl := &http.Client{}

// pass client to witness to wrap its Transport with eavesdropping functionality
// it will also start SSE notification channel
witness.DebugClient(cl)

// then use cl as http client for making request
```

Alternatively, is you are interested in behaviour of some third-party http client, k8s client for example, you could eavesdrop on http client created by k8s go client code.
TODO: make demo of k8s client eavesdropping
