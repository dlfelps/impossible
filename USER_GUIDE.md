# GPS Data Processor - User Guide

## Table of Contents

1. [Overview](#overview)
2. [Installation](#installation)
3. [Data Requirements](#data-requirements)
4. [Configuration](#configuration)
5. [Basic Usage](#basic-usage)
6. [Advanced Usage](#advanced-usage)
7. [Output Files](#output-files)
8. [Troubleshooting](#troubleshooting)
9. [Windows Security Warnings](#windows-security-warnings)

## Overview

GPS Data Processor is a command-line tool for processing and analyzing GPS trajectory data. It reads CSV files containing GPS coordinates, calculates distances and speeds between consecutive points, and outputs processed data in both CSV and KML formats for further analysis and visualization.

### Key Features

- Process GPS tracking data from standard CSV files
- Group data points by device ID
- Calculate distances between consecutive points using the Haversine formula
- Calculate time differences and speeds
- Filter out stationary or slow-moving points
- Generate KML files for visualization in Google Earth and other mapping applications
- Support for custom column mappings via configuration files

## Installation

### Prerequisites

- Windows, macOS, or Linux operating system
- No additional dependencies required (self-contained executable)

### Download and Install

1. Download the latest `gps-processor.exe` (Windows) or `gps-processor` (macOS/Linux) from the releases page
2. Save the executable in a location of your choice
3. No installation is required - the program can be run directly

### Building from Source

If you prefer to build from source:

```bash
# Clone the repository
git clone https://github.com/yourusername/gps-processor.git
cd gps-processor

# Build the executable
go build -o gps-processor
```

## Data Requirements

The GPS Data Processor requires CSV input files with the following characteristics:

### Required CSV Format

- Must include a header row with column names
- Must contain columns for device ID, latitude, longitude, and timestamp
- Timestamps must be in RFC3339 format (e.g., `2023-03-01T12:00:00Z`)
- Additional columns are allowed and will be preserved in the output

### Example Input CSV

```csv
ID,latitude,longitude,timestamp
device1,40.7128,-74.0060,2023-04-01T12:00:00Z
device1,40.7130,-74.0062,2023-04-01T12:01:00Z
device2,37.7749,-122.4194,2023-04-01T12:00:00Z
device2,37.7751,-122.4195,2023-04-01T12:02:00Z
```

## Configuration

The GPS Data Processor can be configured using a YAML configuration file.

### Default Configuration

When run for the first time, the program automatically generates a default `config.yaml` file with the following contents:

```yaml
# GPS Processor Configuration

# CSV Column Mappings (specify the column names in your CSV file)
columns:
  id: "ID"               # Device/track identifier
  latitude: "latitude"   # Latitude coordinate
  longitude: "longitude" # Longitude coordinate  
  timestamp: "timestamp" # Timestamp in RFC3339 format

# Processing Parameters
parameters:
  filter_above_kph: 1.0  # Filter out records with speed below this value (km/h)
```

### Custom Configuration

You can modify the configuration file to match your CSV column names and adjust processing parameters. For example, if your CSV uses different column names:

```yaml
# Custom configuration for a different CSV format
columns:
  id: "deviceID"
  latitude: "lat"
  longitude: "lon"
  timestamp: "time"

parameters:
  filter_above_kph: 3.5  # Increase speed filter threshold to 3.5 km/h
```

## Basic Usage

### Command Syntax

```
gps-processor [input_file] [filter_speed] [config_file]
```

### Simple Examples

Process a CSV file with default settings:
```
gps-processor track_data.csv
```

Process a CSV file and filter out points with speed below 2.5 km/h:
```
gps-processor track_data.csv 2.5
```

Use a custom configuration file:
```
gps-processor track_data.csv my_config.yaml
```

### Auto-detection

If you don't specify input files, the program will automatically use:
- The only CSV file in the current directory (if only one exists)
- The config.yaml file (or the only YAML file if only one exists)

```
gps-processor
```

## Advanced Usage

### Combining Parameters

You can specify both a speed threshold and a configuration file:

```
gps-processor track_data.csv 3.0 custom_config.yaml
```

### Windows Command Prompt Usage

In Windows Command Prompt or PowerShell:

```
gps-processor.exe track_data.csv 2.0
```

### Help and Documentation

To view help information and examples:

```
gps-processor --help
```

or

```
gps-processor -h
```

## Output Files

### CSV Output

The program generates a processed CSV file with the following additional columns:

- `previous_row`: Reference to the row number of the previous point for the same device
- `prev_latitude`: Latitude of the previous point
- `prev_longitude`: Longitude of the previous point
- `prev_timestamp`: Timestamp of the previous point
- `time_diff_seconds`: Time difference between consecutive points in seconds
- `distance_km`: Distance between consecutive points in kilometers
- `speed_kmh`: Speed between consecutive points in kilometers per hour

Output filename: `input_filename_processed.csv`

### KML Output

The program also generates a KML file for visualization in Google Earth or other mapping applications:

- Trajectory lines are color-coded by device ID
- Points include detailed information when clicked
- Device data is organized in folders

Output filename: `input_filename_processed.kml`

## Troubleshooting

### Common Issues

#### "Error reading CSV: unable to open file"
- Verify that the input file exists and is in the correct location
- Check file permissions to ensure the program can read the file

#### "Warning: Error loading config file"
- Verify that the configuration file exists and is properly formatted
- Check for YAML syntax errors in your configuration file

#### "Error parsing timestamp"
- Ensure timestamps in your CSV are in RFC3339 format (e.g., `2023-03-01T12:00:00Z`)
- Check for any malformed timestamp entries in your CSV file

### When All Output Records Are Filtered

If your output file contains no records:
- Your speed filter threshold may be too high - try lowering it
- Check that your timestamps are in the correct order
- Verify that your coordinates contain actual movement

## Windows Security Warnings

When running the GPS Data Processor on Windows, you may see security warnings because the executable is not digitally signed by a verified publisher.

### Handling Windows Defender SmartScreen Warning

1. When you run the executable, you may see a "Windows protected your PC" message
2. Click "More info"
3. Click "Run anyway" after verifying you downloaded the file from a trusted source

### Verifying File Safety

To ensure the file you downloaded is legitimate:
1. Only download from the official repository or website
2. Check the file hash against the published hash on our release page
3. Scan with your antivirus software before running

### Alternative Options

If you are uncomfortable running the executable:
1. Build from source code after reviewing it
2. Run the source directly with Go installed: `go run main.go kml.go`