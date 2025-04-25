package main

import (
        "bufio"
        "encoding/csv"
        "fmt"
        "os"
        "path/filepath"
        "sort"
        "strconv"
        "time"
        
        "gps-processor/haversine"
        "github.com/schollz/progressbar/v3"
)

// Record represents a single GPS data point
type Record struct {
        ID               string
        Latitude         float64
        Longitude        float64
        Timestamp        time.Time
        OriginalRow      int
        TimeDiff         float64 // time difference in seconds
        Distance         float64 // distance in kilometers
        Speed            float64 // speed in kilometers per hour
        PreviousRow      int     // reference to previous row
        PrevLatitude     float64 // latitude of previous point
        PrevLongitude    float64 // longitude of previous point
        PrevTimestamp    time.Time // timestamp of previous point
}

func main() {
        // Check for input file argument and filter_above_kph parameter
        args := os.Args[1:]
        var inputFile string
        var filterAboveKph float64 = 1.0 // Default to 1.0 km/h

        if len(args) > 0 {
                inputFile = args[0]
        } else {
                inputFile = "sample.csv" // Default to sample.csv if no argument provided
        }

        // Check if there's a second argument for filter_above_kph
        if len(args) > 1 {
                var err error
                filterAboveKph, err = strconv.ParseFloat(args[1], 64)
                if err != nil {
                        fmt.Fprintf(os.Stderr, "Error parsing filter_above_kph parameter: %v\n", err)
                        fmt.Fprintf(os.Stderr, "Using default value: %.1f km/h\n", filterAboveKph)
                }
        }

        fmt.Printf("=== GPS Data Processor ===\n")
        fmt.Printf("Input file: %s\n", inputFile)
        fmt.Printf("Speed filter threshold: %.1f km/h\n\n", filterAboveKph)

        // Start timer to track overall processing time
        startTime := time.Now()

        // Read and process the CSV file
        fmt.Println("Step 1: Reading input CSV file...")
        records, err := readCSV(inputFile)
        if err != nil {
                fmt.Fprintf(os.Stderr, "Error reading CSV: %v\n", err)
                os.Exit(1)
        }

        // Group by ID
        fmt.Println("Step 2: Grouping records by ID...")
        groupedRecords := groupByID(records)
        fmt.Printf("Found %d unique device IDs\n\n", len(groupedRecords))

        // Calculate time differences and distances
        fmt.Println("Step 3: Calculating time differences and distances...")
        processedRecords := processGroups(groupedRecords)
        
        // Filter out records with previous_row = 0 and apply speed filter
        fmt.Println("Step 4: Filtering records...")
        filteredRecords := filterRecords(processedRecords, filterAboveKph)
        fmt.Printf("Filtered from %d to %d records\n\n", len(processedRecords), len(filteredRecords))

        // Output to CSV file
        csvOutputFile := getOutputFilename(inputFile, "csv")
        fmt.Println("Step 5: Writing output CSV file...")
        if err := writeOutputCSV(csvOutputFile, filteredRecords); err != nil {
                fmt.Fprintf(os.Stderr, "Error writing output CSV: %v\n", err)
                os.Exit(1)
        }
        
        // Output to KML file
        kmlOutputFile := getOutputFilename(inputFile, "kml")
        fmt.Println("Step 6: Writing output KML file...")
        if err := writeOutputKML(kmlOutputFile, filteredRecords); err != nil {
                fmt.Fprintf(os.Stderr, "Error writing output KML: %v\n", err)
                os.Exit(1)
        }

        // Print summary
        duration := time.Since(startTime).Seconds()
        fmt.Printf("\n=== Processing Summary ===\n")
        fmt.Printf("Total input records: %d\n", len(records))
        fmt.Printf("Records after filtering: %d\n", len(filteredRecords))
        fmt.Printf("Speed filter threshold: %.1f km/h\n", filterAboveKph)
        fmt.Printf("Processing time: %.2f seconds\n", duration)
        fmt.Printf("CSV output file: %s\n", csvOutputFile)
        fmt.Printf("KML output file: %s\n", kmlOutputFile)
        fmt.Printf("=========================\n")
}

// readCSV reads and parses the CSV file
func readCSV(filename string) ([]Record, error) {
        file, err := os.Open(filename)
        if err != nil {
                return nil, fmt.Errorf("unable to open file: %w", err)
        }
        defer file.Close()

        // Count lines to set up the progress bar
        lineCount, err := countLines(filename)
        if err != nil {
                return nil, fmt.Errorf("error counting lines: %w", err)
        }

        // Create progress bar for reading CSV
        bar := progressbar.NewOptions(
                lineCount-1, // Subtract 1 for header
                progressbar.OptionSetDescription("Reading CSV"),
                progressbar.OptionShowCount(),
                progressbar.OptionSetTheme(progressbar.Theme{
                        Saucer:        "=",
                        SaucerHead:    ">",
                        SaucerPadding: " ",
                        BarStart:      "[",
                        BarEnd:        "]",
                }),
        )

        reader := csv.NewReader(file)

        // Read the header
        header, err := reader.Read()
        if err != nil {
                return nil, fmt.Errorf("error reading header: %w", err)
        }

        // Find column indices
        idIdx, latIdx, lonIdx, timestampIdx := -1, -1, -1, -1
        for i, col := range header {
                switch col {
                case "ID":
                        idIdx = i
                case "latitude":
                        latIdx = i
                case "longitude":
                        lonIdx = i
                case "timestamp":
                        timestampIdx = i
                }
        }

        // Validate all required columns exist
        if idIdx == -1 || latIdx == -1 || lonIdx == -1 || timestampIdx == -1 {
                return nil, fmt.Errorf("missing required columns (ID, latitude, longitude, timestamp)")
        }

        var records []Record
        rowNumber := 1 // Starting from 1 to account for header

        // Read the rest of the rows
        for {
                row, err := reader.Read()
                if err != nil {
                        if err.Error() == "EOF" {
                                break
                        }
                        return nil, fmt.Errorf("error reading row: %w", err)
                }
                rowNumber++

                // Update progress bar
                _ = bar.Add(1)

                // Parse latitude and longitude
                lat, err := strconv.ParseFloat(row[latIdx], 64)
                if err != nil {
                        return nil, fmt.Errorf("invalid latitude at row %d: %w", rowNumber, err)
                }
                lon, err := strconv.ParseFloat(row[lonIdx], 64)
                if err != nil {
                        return nil, fmt.Errorf("invalid longitude at row %d: %w", rowNumber, err)
                }

                // Parse timestamp
                ts, err := time.Parse(time.RFC3339, row[timestampIdx])
                if err != nil {
                        return nil, fmt.Errorf("invalid timestamp at row %d: %w", rowNumber, err)
                }

                // Create record
                records = append(records, Record{
                        ID:          row[idIdx],
                        Latitude:    lat,
                        Longitude:   lon,
                        Timestamp:   ts,
                        OriginalRow: rowNumber,
                })
        }

        fmt.Println() // Add newline after progress bar
        return records, nil
}

// countLines counts the number of lines in a file
func countLines(filename string) (int, error) {
        file, err := os.Open(filename)
        if err != nil {
                return 0, err
        }
        defer file.Close()

        scanner := bufio.NewScanner(file)
        lineCount := 0
        for scanner.Scan() {
                lineCount++
        }

        if err := scanner.Err(); err != nil {
                return 0, err
        }

        return lineCount, nil
}

// groupByID groups records by ID
func groupByID(records []Record) map[string][]Record {
        groups := make(map[string][]Record)
        for _, record := range records {
                groups[record.ID] = append(groups[record.ID], record)
        }
        return groups
}

// processGroups sorts each group by timestamp and calculates time differences and distances
func processGroups(groups map[string][]Record) []Record {
        var processedRecords []Record
        
        // Calculate total number of records to process for the progress bar
        totalRecords := 0
        for _, group := range groups {
                totalRecords += len(group)
        }
        
        // Create progress bar for processing
        bar := progressbar.NewOptions(
                totalRecords,
                progressbar.OptionSetDescription("Processing GPS data"),
                progressbar.OptionShowCount(),
                progressbar.OptionSetTheme(progressbar.Theme{
                        Saucer:        "=",
                        SaucerHead:    ">",
                        SaucerPadding: " ",
                        BarStart:      "[",
                        BarEnd:        "]",
                }),
        )

        for _, group := range groups {
                // Sort by timestamp
                sort.Slice(group, func(i, j int) bool {
                        return group[i].Timestamp.Before(group[j].Timestamp)
                })

                // Calculate time differences and distances
                for i := 0; i < len(group); i++ {
                        // Update progress bar
                        _ = bar.Add(1)
                        
                        if i > 0 {
                                // Calculate time difference
                                timeDiff := group[i].Timestamp.Sub(group[i-1].Timestamp).Seconds()
                                
                                // Calculate haversine distance
                                distance := haversine.Distance(
                                        group[i-1].Latitude, group[i-1].Longitude,
                                        group[i].Latitude, group[i].Longitude,
                                )

                                group[i].TimeDiff = timeDiff
                                group[i].Distance = distance
                                group[i].PreviousRow = group[i-1].OriginalRow
                                
                                // Calculate speed in kilometers per hour
                                // Speed = (distance in km) / (time in hours)
                                // timeDiff is in seconds, so convert to hours by dividing by 3600
                                if timeDiff > 0 {
                                        group[i].Speed = distance / (timeDiff / 3600)
                                } else {
                                        group[i].Speed = 0
                                }
                                
                                // Store previous point's data
                                group[i].PrevLatitude = group[i-1].Latitude
                                group[i].PrevLongitude = group[i-1].Longitude
                                group[i].PrevTimestamp = group[i-1].Timestamp
                        } else {
                                // First record in the group has no previous point
                                group[i].TimeDiff = 0
                                group[i].Distance = 0
                                group[i].Speed = 0
                                group[i].PreviousRow = 0
                                // Set previous point data to zero values
                                group[i].PrevLatitude = 0
                                group[i].PrevLongitude = 0
                                // Leave PrevTimestamp as zero value (1970-01-01 00:00:00 +0000 UTC)
                        }
                        processedRecords = append(processedRecords, group[i])
                }
        }

        fmt.Println() // Add newline after progress bar
        return processedRecords
}

// filterRecords removes records with previous_row = 0 and optionally filters by speed threshold
func filterRecords(records []Record, filterAboveKph float64) []Record {
        // Create a progress bar for filtering
        bar := progressbar.NewOptions(
                len(records),
                progressbar.OptionSetDescription("Filtering records"),
                progressbar.OptionShowCount(),
                progressbar.OptionSetTheme(progressbar.Theme{
                        Saucer:        "=",
                        SaucerHead:    ">",
                        SaucerPadding: " ",
                        BarStart:      "[",
                        BarEnd:        "]",
                }),
        )

        var filtered []Record
        var speedFilteredCount int
        
        for _, record := range records {
                // Update progress bar
                _ = bar.Add(1)
                
                // Only keep records with previous_row not equal to 0
                if record.PreviousRow != 0 {
                        // Apply speed filtering
                        if record.Speed >= filterAboveKph {
                                filtered = append(filtered, record)
                        } else {
                                speedFilteredCount++
                        }
                }
        }
        
        fmt.Println() // Add newline after progress bar
        if filterAboveKph > 0 {
                fmt.Printf("Speed filter applied: Removed %d records with speed below %.1f km/h\n", 
                        speedFilteredCount, filterAboveKph)
        }
        return filtered
}

// getOutputFilename generates the output filename
func getOutputFilename(inputFile string, format string) string {
        ext := filepath.Ext(inputFile)
        baseName := inputFile[:len(inputFile)-len(ext)]
        
        if format == "kml" {
                return baseName + "_processed.kml"
        }
        
        // Default to CSV format
        return baseName + "_processed.csv"
}

// writeOutputKML writes the processed records to a KML file for visualization
func writeOutputKML(filename string, records []Record) error {
        file, err := os.Create(filename)
        if err != nil {
                return fmt.Errorf("unable to create KML file: %w", err)
        }
        defer file.Close()
        
        // Group records by ID
        groups := make(map[string][]Record)
        for _, record := range records {
                groups[record.ID] = append(groups[record.ID], record)
        }
        
        // Create progress bar for KML generation
        bar := progressbar.NewOptions(
                len(groups),
                progressbar.OptionSetDescription("Writing output KML"),
                progressbar.OptionShowCount(),
                progressbar.OptionSetTheme(progressbar.Theme{
                        Saucer:        "=",
                        SaucerHead:    ">",
                        SaucerPadding: " ",
                        BarStart:      "[",
                        BarEnd:        "]",
                }),
        )
        
        // XML header
        fmt.Fprintln(file, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>")
        fmt.Fprintln(file, "<kml xmlns=\"http://www.opengis.net/kml/2.2\">")
        fmt.Fprintln(file, "<Document>")
        fmt.Fprintln(file, "  <name>GPS Trajectories</name>")
        fmt.Fprintln(file, "  <description>GPS data processed by GPS Processor</description>")
        
        // Add styles for different IDs
        // Using a simple hash function to generate different colors for different IDs
        fmt.Fprintln(file, "  <Style id=\"defaultStyle\">")
        fmt.Fprintln(file, "    <LineStyle>")
        fmt.Fprintln(file, "      <color>ff0000ff</color>") // Red
        fmt.Fprintln(file, "      <width>4</width>")
        fmt.Fprintln(file, "    </LineStyle>")
        fmt.Fprintln(file, "    <IconStyle>")
        fmt.Fprintln(file, "      <color>ff0000ff</color>") // Red
        fmt.Fprintln(file, "      <scale>0.5</scale>")
        fmt.Fprintln(file, "    </IconStyle>")
        fmt.Fprintln(file, "  </Style>")
        
        // Define some common colors
        colors := []string{
                "ff0000ff", // Red
                "ff00ff00", // Green
                "ffff0000", // Blue
                "ff00ffff", // Yellow
                "ffff00ff", // Magenta
        }
        
        // Create a folder for each ID
        idCount := 0
        for id, group := range groups {
                // Update progress bar
                _ = bar.Add(1)
                
                // Sort by timestamp to ensure correct order
                sort.Slice(group, func(i, j int) bool {
                        return group[i].Timestamp.Before(group[j].Timestamp)
                })
                
                // Generate a color based on the ID
                colorIndex := idCount % len(colors)
                color := colors[colorIndex]
                idCount++
                
                // Create a unique style for this ID
                styleID := fmt.Sprintf("style_%s", id)
                fmt.Fprintf(file, "  <Style id=\"%s\">\n", styleID)
                fmt.Fprintln(file, "    <LineStyle>")
                fmt.Fprintf(file, "      <color>%s</color>\n", color)
                fmt.Fprintln(file, "      <width>4</width>")
                fmt.Fprintln(file, "    </LineStyle>")
                fmt.Fprintln(file, "    <IconStyle>")
                fmt.Fprintf(file, "      <color>%s</color>\n", color)
                fmt.Fprintln(file, "      <scale>0.5</scale>")
                fmt.Fprintln(file, "    </IconStyle>")
                fmt.Fprintln(file, "  </Style>")
                
                // Create a folder for this ID
                fmt.Fprintf(file, "  <Folder>\n")
                fmt.Fprintf(file, "    <name>Device %s</name>\n", id)
                
                // Create a placemark for the trajectory
                fmt.Fprintln(file, "    <Placemark>")
                fmt.Fprintf(file, "      <name>Trajectory of Device %s</name>\n", id)
                fmt.Fprintln(file, "      <description><![CDATA[")
                fmt.Fprintf(file, "Number of points: %d<br>\n", len(group))
                fmt.Fprintf(file, "Start time: %s<br>\n", group[0].Timestamp.Format(time.RFC3339))
                fmt.Fprintf(file, "End time: %s<br>\n", group[len(group)-1].Timestamp.Format(time.RFC3339))
                fmt.Fprintln(file, "      ]]></description>")
                fmt.Fprintf(file, "      <styleUrl>#%s</styleUrl>\n", styleID)
                fmt.Fprintln(file, "      <LineString>")
                fmt.Fprintln(file, "        <extrude>1</extrude>")
                fmt.Fprintln(file, "        <tessellate>1</tessellate>")
                fmt.Fprintln(file, "        <altitudeMode>clampToGround</altitudeMode>")
                fmt.Fprintln(file, "        <coordinates>")
                
                // Add all coordinates for the trajectory
                for _, record := range group {
                        fmt.Fprintf(file, "          %f,%f,0\n", record.Longitude, record.Latitude)
                }
                
                fmt.Fprintln(file, "        </coordinates>")
                fmt.Fprintln(file, "      </LineString>")
                fmt.Fprintln(file, "    </Placemark>")
                
                // Create individual placemarks for each point with detailed information
                for i, record := range group {
                        fmt.Fprintln(file, "    <Placemark>")
                        fmt.Fprintf(file, "      <name>Point %d (Device %s)</name>\n", i+1, id)
                        fmt.Fprintln(file, "      <description><![CDATA[")
                        fmt.Fprintf(file, "ID: %s<br>\n", record.ID)
                        fmt.Fprintf(file, "Latitude: %f<br>\n", record.Latitude)
                        fmt.Fprintf(file, "Longitude: %f<br>\n", record.Longitude)
                        fmt.Fprintf(file, "Timestamp: %s<br>\n", record.Timestamp.Format(time.RFC3339))
                        fmt.Fprintf(file, "Original Row: %d<br>\n", record.OriginalRow)
                        fmt.Fprintf(file, "Previous Row: %d<br>\n", record.PreviousRow)
                        if record.PreviousRow > 0 {
                                fmt.Fprintf(file, "Previous Latitude: %f<br>\n", record.PrevLatitude)
                                fmt.Fprintf(file, "Previous Longitude: %f<br>\n", record.PrevLongitude)
                                fmt.Fprintf(file, "Previous Timestamp: %s<br>\n", record.PrevTimestamp.Format(time.RFC3339))
                                fmt.Fprintf(file, "Time Difference: %.2f seconds<br>\n", record.TimeDiff)
                                fmt.Fprintf(file, "Distance: %.6f km<br>\n", record.Distance)
                                fmt.Fprintf(file, "Speed: %.2f km/h<br>\n", record.Speed)
                        }
                        fmt.Fprintln(file, "      ]]></description>")
                        fmt.Fprintf(file, "      <styleUrl>#%s</styleUrl>\n", styleID)
                        fmt.Fprintln(file, "      <Point>")
                        fmt.Fprintln(file, "        <coordinates>")
                        fmt.Fprintf(file, "          %f,%f,0\n", record.Longitude, record.Latitude)
                        fmt.Fprintln(file, "        </coordinates>")
                        fmt.Fprintln(file, "      </Point>")
                        fmt.Fprintln(file, "    </Placemark>")
                }
                
                fmt.Fprintln(file, "  </Folder>")
        }
        
        // Close XML document
        fmt.Fprintln(file, "</Document>")
        fmt.Fprintln(file, "</kml>")
        
        fmt.Println() // Add newline after progress bar
        return nil
}

// writeOutputCSV writes the processed records to a new CSV file
func writeOutputCSV(filename string, records []Record) error {
        file, err := os.Create(filename)
        if err != nil {
                return fmt.Errorf("unable to create output file: %w", err)
        }
        defer file.Close()

        writer := csv.NewWriter(file)
        defer writer.Flush()

        // Write header with additional columns for previous point data
        header := []string{
                "ID", 
                "latitude", 
                "longitude", 
                "timestamp", 
                "original_row", 
                "previous_row", 
                "prev_latitude", 
                "prev_longitude", 
                "prev_timestamp",
                "time_diff_seconds", 
                "distance_km",
                "speed_kmh",
        }
        if err := writer.Write(header); err != nil {
                return fmt.Errorf("error writing header: %w", err)
        }

        // Create progress bar for writing CSV
        bar := progressbar.NewOptions(
                len(records),
                progressbar.OptionSetDescription("Writing output CSV"),
                progressbar.OptionShowCount(),
                progressbar.OptionSetTheme(progressbar.Theme{
                        Saucer:        "=",
                        SaucerHead:    ">",
                        SaucerPadding: " ",
                        BarStart:      "[",
                        BarEnd:        "]",
                }),
        )

        // Write data
        for _, record := range records {
                // Format previous timestamp, handle zero value
                prevTimestampStr := ""
                if !record.PrevTimestamp.IsZero() {
                        prevTimestampStr = record.PrevTimestamp.Format(time.RFC3339)
                }
                
                row := []string{
                        record.ID,
                        fmt.Sprintf("%f", record.Latitude),
                        fmt.Sprintf("%f", record.Longitude),
                        record.Timestamp.Format(time.RFC3339),
                        fmt.Sprintf("%d", record.OriginalRow),
                        fmt.Sprintf("%d", record.PreviousRow),
                        fmt.Sprintf("%f", record.PrevLatitude),
                        fmt.Sprintf("%f", record.PrevLongitude),
                        prevTimestampStr,
                        fmt.Sprintf("%f", record.TimeDiff),
                        fmt.Sprintf("%f", record.Distance),
                        fmt.Sprintf("%f", record.Speed),
                }
                if err := writer.Write(row); err != nil {
                        return fmt.Errorf("error writing row: %w", err)
                }
                
                // Update progress bar
                _ = bar.Add(1)
        }

        fmt.Println() // Add newline after progress bar
        return nil
}
