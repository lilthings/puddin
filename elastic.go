package main

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/olivere/elastic"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

func getElasticClient() *elastic.Client {
	var client *elastic.Client
	if os.Getenv("PUDDIN_HTTPS") == "true" {
		httpClient, err := getHttpClient()
		if err != nil {
			panic(err)
		}

		client, err = elastic.NewClient(elastic.SetSniff(false), elastic.SetHttpClient(httpClient),
			elastic.SetURL("https://127.0.0.1:9200"))

		if err != nil {
			panic(err)
		}
	} else {
		var err error
		client, err = elastic.NewClient(elastic.SetURL("http://127.0.0.1:9200"))
		if err != nil {
			panic(err)
		}
	}

	return client
}

func getHttpClient() (*http.Client, error) {
	var httpClient *http.Client

	certFile := "config/cert/client.pem"
	certKey := "config/cert/client.key"
	rootCertPath := "config/cert/root-ca.pem"

	cert, err := tls.LoadX509KeyPair(certFile, certKey)
	if err != nil {
		return nil, err
	}
	caCert, err := ioutil.ReadFile(rootCertPath)
	if err != nil {
		return nil, err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// Setup HTTPS client
	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		RootCAs:            caCertPool,
		InsecureSkipVerify: true,
	}
	tlsConfig.BuildNameToCertificate()
	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	httpClient = &http.Client{
		Transport: transport,
	}

	return httpClient, nil
}

const mapping = ``

func createOnlineRoomIndex() {
	// // Use the IndexExists service to check if a specified index exists.
	// exists, err := client.IndexExists("online_rooms").Do(ctx)
	// if err != nil {
	// 	// Handle error
	// 	panic(err)
	// }
	// if !exists {
	// 	// Create a new index.
	// 	createIndex, err := client.CreateIndex("online_rooms").BodyString(mapping).Do(ctx)
	// 	if err != nil {
	// 		// Handle error
	// 		panic(err)
	// 	}
	// 	if !createIndex.Acknowledged {
	// 		// Not acknowledged
	// 	}
	// }
}

type elasticOM struct {
	Time  time.Time   `json:"time"`
	Model OnlineModel `json:"model"`
}
