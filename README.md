# ParamProbe

ParamProbe is a tool designed to discover reflected parameters on websites. This tool is useful for identifying URL parameters that are vulnerable to attacks such as Reflected XSS.

## Features

- **Automatic Crawling**: Explores websites and collects all URL parameters.
- **Reflected Parameter Detection**: Tests parameters reflected in HTTP responses.
- **Crawling Subdomain**: Option to explore subdomains.
- **Custom User-Agent**: Supports custom or random User-Agent.
- **Automatic Updates**: Provides support to update the tool to the latest version.

## Installation
```
go install github.com/fkr00t/paramprobe@latest
```

## Usage
```
paramprobe -h
```

This will display help for the tool. Here are all the switches it supports.

```
Option:
    -u, --url           Target URL to scan.
    -c, --crawl         Crawl subdomains.
    -d, --delay         Delay between requests (e.g., 1s).
    --user-agent        Custom User-Agent.
    --random-agent      Use a random User-Agent.
    -up, --update       Update the tool to the latest version.
    -h, --help          Show help message.


example:
    paramprobe -u http://testphp.vulnweb.com -d 1s --random-agent
    paramprobe -u http://testphp.vulnweb.com --user-agent 'MyCustomAgent'
    paramprobe --update
```