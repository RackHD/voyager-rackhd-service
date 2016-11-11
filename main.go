package main

import (
	"flag"
	"log"

	"github.com/RackHD/voyager-rackhd-service/server"
)

var (
	uri           = flag.String("uri", "amqp://guest:guest@rabbitmq:5672/", "AMQP URI")
	rackhdAddress = flag.String("rackhd-address", "http://192.168.50.4:8080", "RACKHD URI")
)

func init() {
	flag.Parse()
}

func main() {

	server := server.NewServer(*uri, *rackhdAddress)
	if server.MQ == nil {
		log.Fatalf("Could not connect to RabbitMQ")
	}
	defer server.MQ.Close()

	server.Start()

	select {}
}
