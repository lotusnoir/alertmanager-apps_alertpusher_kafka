package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/optiopay/kafka/v2"
	"github.com/vma/getopt"
	"github.com/vma/glog"
	"github.com/vma/httplogger"
)

var (
	// Revision is the git revision, set at compilation
	Revision string

	// Build is the build time, set at compilation
	Build string

	// Branch is the git branch, set at compilation
	Branch string

	kafkaHosts   = getopt.ListLong("kafka-hosts", 'k', "kafka brokers")
	kafkaTopic   = getopt.StringLong("topic", 't', "alert.noc", "kafka topic for the alerts")
	kafkaPart    = getopt.Int32Long("partition", 0, 0, "kafka partition")
	port         = getopt.IntLong("port", 'p', 8086, "web server listen port", "port")
	debug        = getopt.IntLong("debug", 'd', 0, "debug level")
	logDir       = getopt.StringLong("log", 0, "", "directory for log files, all log goes to stderr if empty", "dir")
	showVersion  = getopt.BoolLong("version", 'v', "Prints version and build date")
	skipResolved = getopt.BoolLong("skip-resolved", 's', "Skips messages for resolved alerts")

	producer kafka.Producer
)

func main() {
	getopt.SetParameters("")
	getopt.Parse()
	glog.WithConf(glog.Conf{Verbosity: *debug, LogDir: *logDir, PrintLocation: *debug > 0})
	http.HandleFunc("/alert", HandleAlert)

	if *showVersion {
		fmt.Printf("Revision:%s Branch:%s Build:%s\n", Revision, Branch, Build)
		return
	}

	if len(*kafkaHosts) == 0 {
		glog.Exit("kafka hosts missing")
	}

	for i, h := range *kafkaHosts {
		if strings.LastIndex(h, ":") == -1 {
			(*kafkaHosts)[i] += ":9092"
		}
	}

	brokerConf := kafka.NewBrokerConf(fmt.Sprintf("alertpusher-kafka[%d]", os.Getpid()))
	brokerConf.ReadTimeout = 0
	glog.V(2).Info("dialing kafka...")
	broker, err := kafka.Dial(*kafkaHosts, brokerConf)
	if err != nil {
		glog.Exitf("kafka dial: %v", err)
	}
	defer broker.Close()
	glog.V(1).Info("connected to kafka")
	producer = broker.Producer(kafka.NewProducerConf())

	logger := httplogger.CommonLogger(os.Stderr)
	glog.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), logger(http.DefaultServeMux)))
}
