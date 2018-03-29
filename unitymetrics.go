package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/equelin/gounity"
	"github.com/sirupsen/logrus"
)

var log = logrus.New()
var unityName string

//
func parseResult(timestamp time.Time, path string, valuesMap map[string]interface{}) {

	tagsMap := make(map[string]string)
	tagNames := make(map[int]string)

	pathSplit := strings.Split(path, ".")

	var measurementName string
	if pathSplit[0] == "kpi" {
		measurementName = fmt.Sprintf("kpi_%s", pathSplit[1])
	} else {
		measurementName = pathSplit[2]
	}

	j := 0
	for i, v := range pathSplit {
		if v == "*" {
			tagName := pathSplit[i-1]
			tagNames[j] = tagName
			j++
		}
	}

	parseMap(
		timestamp,
		0,
		&path,
		&measurementName,
		tagNames,
		tagsMap,
		valuesMap,
	)
}

// https://stackoverflow.com/questions/29366038/looping-iterate-over-the-second-level-nested-json-in-go-lang
func parseMap(timestamp time.Time, index int, pathPtr *string, measurementNamePtr *string, tagNames map[int]string, tagsMap map[string]string, valuesMap map[string]interface{}) {

	for key, val := range valuesMap {

		pathSplit := strings.Split(*pathPtr, ".")

		switch concreteVal := val.(type) {
		case map[string]interface{}:

			ok := false

			for i, v := range pathSplit {
				if v == key {
					tagName := pathSplit[i-1]
					tagsMap[tagName] = key
					ok = true
				}
			}

			if ok != true {
				tagsMap[tagNames[index]] = key
				index++
			}

			parseMap(
				timestamp,
				index,
				pathPtr,
				measurementNamePtr,
				tagNames,
				tagsMap,
				val.(map[string]interface{}),
			)

		default:

			if len(tagNames) != 0 {
				tagsMap[tagNames[index]] = key
			} else {
				for i, v := range pathSplit {
					if v == key {
						tagName := pathSplit[i-1]
						tagsMap[tagName] = key
					}
				}
			}

			// Formating tags set
			// <tag_key>=<tag_value>,<tag_key>=<tag_value>
			var tags string

			tags = fmt.Sprintf("unity=%s", unityName)
			for k, v := range tagsMap {
				tags = tags + fmt.Sprintf(",%s=%s", k, v)
			}

			// Formating fied set
			// <field_key>=<field_value>
			var field string
			_, ok := concreteVal.(float64)

			if ok {
				field = fmt.Sprintf("%s=%f", pathSplit[len(pathSplit)-1], concreteVal)
			} else {
				field = fmt.Sprintf("%s=%s", pathSplit[len(pathSplit)-1], concreteVal)
			}

			// Formating and printing the result using the InfluxDB's Line Protocol
			// https://docs.influxdata.com/influxdb/v1.5/write_protocols/line_protocol_tutorial/
			fmt.Printf("%s,%s %s %d\n", *measurementNamePtr, tags, field, timestamp.UnixNano())
		}
	}
}

func main() {

	// Set logs parameters
	log.Out = os.Stdout

	userPtr := flag.String("user", "", "Username")
	passwordPtr := flag.String("password", "", "Password")
	unityPtr := flag.String("unity", "", "Unity IP or FQDN")
	intervalPtr := flag.Uint64("interval", 30, "Sampling interval")
	rtpathsPtr := flag.String("rtpaths", "", "Real time metrics paths")
	histpathsPtr := flag.String("histpaths", "", "Historical metrics paths")
	debugPtr := flag.Bool("debug", false, "Debug mode")

	flag.Parse()

	if *debugPtr == true {
		log.Level = logrus.DebugLevel
	} else {
		log.Level = logrus.ErrorLevel
	}

	log.WithFields(logrus.Fields{
		"event": "flag",
		"key":   "user",
		"value": *userPtr,
	}).Debug("Parsed flag user")

	log.WithFields(logrus.Fields{
		"event": "flag",
		"key":   "unity",
		"value": *unityPtr,
	}).Debug("Parsed flag unity")

	log.WithFields(logrus.Fields{
		"event": "flag",
		"key":   "interval",
		"value": *intervalPtr,
	}).Debug("Parsed flag interval")

	log.WithFields(logrus.Fields{
		"event": "flag",
		"key":   "paths",
		"value": *rtpathsPtr,
	}).Debug("Parsed flag real time metrics paths")

	log.WithFields(logrus.Fields{
		"event": "flag",
		"key":   "paths",
		"value": *histpathsPtr,
	}).Debug("Parsed flag historical metrics paths")

	// Start a new Unity session

	log.WithFields(logrus.Fields{
		"event":       "gounity.NewSession",
		"unity":       *unityPtr,
		"engineering": "true",
		"user":        *userPtr,
	}).Debug("Started new Unity session")

	session, err := gounity.NewSession(*unityPtr, true, *userPtr, *passwordPtr)

	if err != nil {
		log.Fatal(err)
	}

	defer session.CloseSession()

	// Get system informations
	System, err := session.GetbasicSystemInfo()
	if err != nil {
		log.Fatal(err)
	}

	// Store the name of the Unity
	unityName = System.Entries[0].Content.Name

	if *histpathsPtr != "" {

		// metric paths
		histpaths := strings.Split(*histpathsPtr, ",")

		for _, p := range histpaths {

			log.WithFields(logrus.Fields{
				"event": "historical",
				"key":   "paths",
				"value": p,
			}).Debug("Querying historical metric")

			// Request a new metric query
			MetricValue, err := session.GetmetricValue(p)
			if err != nil {
				log.WithFields(logrus.Fields{
					"event": "historical",
					"key":   "paths",
					"value": p,
					"error": err,
				}).Error("Querying historical metric")
			} else {
				parseResult(MetricValue.Entries[0].Content.Timestamp, MetricValue.Entries[0].Content.Path, MetricValue.Entries[0].Content.Values.(map[string]interface{}))
			}
		}
	}

	if *rtpathsPtr != "" {

		// metric paths
		rtpaths := strings.Split(*rtpathsPtr, ",")

		// converting metric interval into uint32
		var interval = uint32(*intervalPtr)

		// Request a new metric query
		Metric, err := session.NewMetricRealTimeQuery(rtpaths, interval)
		if err != nil {
			log.Fatal(err)
		}

		// Waiting that the sampling of the metrics is done
		time.Sleep(time.Duration(Metric.Content.Interval) * time.Second)

		// Get the results of the query
		Result, err := session.GetMetricRealTimeQueryResult(Metric.Content.ID)
		if err != nil {
			log.WithFields(logrus.Fields{
				"event": "realtime",
				"key":   "error",
				"error": err,
			}).Error("Querying historical metric")
		} else {
			// Parse the results
			for _, v := range Result.Entries {

				parseResult(v.Content.Timestamp, v.Content.Path, v.Content.Values.(map[string]interface{}))
			}
		}
	}
}
