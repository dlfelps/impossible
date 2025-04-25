package main

import (
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/schollz/progressbar/v3"
)

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
