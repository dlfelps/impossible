# GPS Data Processor

A Golang CLI tool that processes GPS data from CSV files, calculates time differences and haversine distances between consecutive points, and outputs results to a new CSV file.

## Features

- Reads CSV files with GPS data (latitude, longitude, timestamp, ID)
- Groups GPS points by ID
- Sorts points by timestamp within each group
- Calculates time differences between consecutive points
- Calculates haversine distances between consecutive points
- Outputs processed data to a new CSV file with references to original row numbers

## Installation

Clone this repository and build the Go application:

```bash
go build -o gps-processor
```

Or to build a Windows executable:

```bash
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o gps-processor.exe main.go kml.go
```