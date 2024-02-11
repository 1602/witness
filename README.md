[![Go Report Card](https://goreportcard.com/badge/github.com/1602/witness)](https://goreportcard.com/report/github.com/1602/witness)
[![Build Status](https://travis-ci.org/1602/witness.svg?branch=master)](https://travis-ci.org/1602/witness)
[![Coverage Status](https://img.shields.io/coveralls/github/1602/witness.svg)](https://coveralls.io/github/1602/witness?branch=master)

## Witness

Enables debugging of http requests via UI. It is like chrome devtools for go backend.

## How it works

It uses `httptrace.WithClientTrace` to make a `http.RoundTripper` eavesdropping on http connection. This allows detailed analysis of various http request stages. All information gathered then pushed using EventStream to UI running in browser.

## Usage

```
// init http client with any configuration you need
cl := &http.Client{}

// pass client to witness to wrap its Transport with eavesdropping functionality
// it will also start SSE notification channel
witness.DebugClient(cl)
```

Open SSE client in browser https://1602.github.io/puerh/
