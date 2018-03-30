package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/equelin/gounity"
	"github.com/sirupsen/logrus"
)

// Types
type pool struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	SizeFree       uint64 `json:"sizeFree"`
	SizeTotal      uint64 `json:"sizeTotal"`
	SizeUsed       uint64 `json:"sizeUsed"`
	SizeSubscribed uint64 `json:"sizeSubscribed"`
}

type storageresource struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	SizeAllocated uint64 `json:"sizeAllocated"`
	SizeTotal     uint64 `json:"sizeTotal"`
	SizeUsed      uint64 `json:"sizeUsed"`
	Type          int    `json:"type"`
}

// Variables
var log = logrus.New()
var unityName string
var unityPools []pool
var unityStorageResource []storageresource

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
			tagsMap["unity"] = unityName

			// Formating fied set
			// <field_key>=<field_value>
			fieldsMap := make(map[string]string)
			_, ok := concreteVal.(float64)

			if ok {
				fieldsMap[pathSplit[len(pathSplit)-1]] = fmt.Sprintf("%f", concreteVal)
			} else {
				fieldsMap[pathSplit[len(pathSplit)-1]] = fmt.Sprintf("%s", concreteVal)
			}

			// Formating and printing the result using the InfluxDB's Line Protocol
			// https://docs.influxdata.com/influxdb/v1.5/write_protocols/line_protocol_tutorial/

			printInflux(*measurementNamePtr, tagsMap, fieldsMap, timestamp.UnixNano())
		}
	}
}

func parsePool(id string, name string, sizeFree uint64, sizeSubscribed uint64, sizeTotal uint64, sizeUsed uint64) {

	tagsMap := make(map[string]string)
	fieldsMap := make(map[string]string)

	tagsMap["unity"] = unityName
	tagsMap["pool"] = id
	tagsMap["poolname"] = name

	fieldsMap["sizefree"] = strconv.FormatUint(sizeFree, 10)
	fieldsMap["sizesubscribed"] = strconv.FormatUint(sizeSubscribed, 10)
	fieldsMap["sizetotal"] = strconv.FormatUint(sizeTotal, 10)
	fieldsMap["sizeused"] = strconv.FormatUint(sizeUsed, 10)

	printInflux("pool", tagsMap, fieldsMap, time.Now().UnixNano())
}

func parseStorageResource(id string, name string, sizeAllocated uint64, sizeTotal uint64, sizeUsed uint64) {

	tagsMap := make(map[string]string)
	fieldsMap := make(map[string]string)

	tagsMap["unity"] = unityName
	tagsMap["storageresource"] = id
	tagsMap["storageresourcename"] = name

	fieldsMap["sizeallocated"] = strconv.FormatUint(sizeAllocated, 10)
	fieldsMap["sizetotal"] = strconv.FormatUint(sizeTotal, 10)
	fieldsMap["sizeused"] = strconv.FormatUint(sizeUsed, 10)

	printInflux("storageresource", tagsMap, fieldsMap, time.Now().UnixNano())
}

func parseKpiValue(id string, name string, path string, value float64) {

	pathSplit := strings.Split(path, ".")
	tagsMap := make(map[string]string)
	fieldsMap := make(map[string]string)

	tagsMap["unity"] = unityName

	for i, v := range pathSplit {
		if v == id {
			tagName := strings.ToLower(pathSplit[i-1])
			tagsMap[tagName] = v
			tagsMap[tagName+"name"] = strings.Replace(name, " ", "_", -1)
		}
		if v == "sp" || v == "rw" || v == "lun" {
			tagsMap["lun"] = pathSplit[i+1]
		}
	}

	fieldsMap[pathSplit[len(pathSplit)-1]] = fmt.Sprintf("%f", value)

	printInflux("kpi_"+pathSplit[1], tagsMap, fieldsMap, time.Now().UnixNano())
}

// printInflux purpose is to output data in the influxdb line format
func printInflux(measurement string, tagsMap map[string]string, fieldsMap map[string]string, timestamp int64) {

	// Parse tagsMap
	var tags string
	var i int
	for k, v := range tagsMap {
		if i == 0 {
			tags = tags + fmt.Sprintf("%s=%s", k, v)
		} else {
			tags = tags + fmt.Sprintf(",%s=%s", k, v)
		}
		i++
	}

	// Parse fieldsMap
	var fields string
	var j int
	for k, v := range fieldsMap {
		if j == 0 {
			fields = fields + fmt.Sprintf("%s=%s", k, v)
		} else {
			fields = fields + fmt.Sprintf(",%s=%s", k, v)
		}
		j++
	}

	fmt.Printf("%s,%s %s %d\n", measurement, tags, fields, timestamp)
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
	histkpipathsPtr := flag.String("histkpipaths", "", "Historical KPI metrics paths")
	capacityPtr := flag.Bool("capacity", false, "Display capacity statisitcs")
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

	log.WithFields(logrus.Fields{
		"event": "flag",
		"key":   "paths",
		"value": *histkpipathsPtr,
	}).Debug("Parsed flag historical KPI metrics paths")

	log.WithFields(logrus.Fields{
		"event": "flag",
		"key":   "capacity",
		"value": *capacityPtr,
	}).Debug("Parsed flag capacity")

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
	} else {
		// Store the name of the Unity
		unityName = System.Entries[0].Content.Name
	}

	// Store pools informations
	Pools, err := session.GetPool()
	if err != nil {
		log.Fatal(err)
	} else {
		for _, p := range Pools.Entries {
			unityPools = append(unityPools, p.Content)
		}
	}

	// Store storage resources informations
	StorageResources, err := session.GetStorageResource()
	if err != nil {
		log.Fatal(err)
	} else {
		for _, s := range StorageResources.Entries {
			unityStorageResource = append(unityStorageResource, s.Content)
		}
	}

	if *histkpipathsPtr != "" {
		// Request a new kpi query
		histkpipaths := strings.Split(*histkpipathsPtr, ",")

		for _, p := range histkpipaths {

			KpiValue, err := session.GetkpiValue(p)
			if err != nil {
				log.WithFields(logrus.Fields{
					"event": "historical",
					"key":   "paths",
					"value": p,
					"error": err,
				}).Error("Querying kpi historical metric(s)")
			} else {
				for _, k := range KpiValue.Entries {
					parseKpiValue(k.Content.ID, k.Content.Name, k.Content.Path, k.Content.Values[k.Content.EndTime])
				}

			}
		}
	}

	if *capacityPtr {
		// Parse pool info into influxdb line protocol
		for _, p := range unityPools {
			parsePool(p.ID, p.Name, p.SizeFree, p.SizeSubscribed, p.SizeTotal, p.SizeUsed)
		}

		// Parse storage resources info into influxdb line protocol
		for _, s := range unityStorageResource {
			parseStorageResource(s.ID, s.Name, s.SizeAllocated, s.SizeTotal, s.SizeUsed)
		}
	}

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
				}).Error("Querying historical metric(s)")
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

		// Waiting thforat the sampling of the metrics to be done
		time.Sleep(time.Duration(Metric.Content.Interval) * time.Second)

		// Get the results of the query
		Result, err := session.GetMetricRealTimeQueryResult(Metric.Content.ID)
		if err != nil {
			log.WithFields(logrus.Fields{
				"event": "realtime",
				"key":   "error",
				"error": err,
			}).Error("Querying real time metric(s)")
		} else {
			// Parse the results
			for _, v := range Result.Entries {

				parseResult(v.Content.Timestamp, v.Content.Path, v.Content.Values.(map[string]interface{}))
			}
		}
	}
}
