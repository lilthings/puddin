package main

import (
	"context"
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

const mapping = `{
  "settings": {
    "number_of_shards": 2
  },
  "mappings": {
    "_doc": {
      "properties": {
        "model": {
          "properties": {
            "age": {
              "type": "long"
            },
            "birthday": {
              "type": "date"
            },
            "chat_room_url": {
              "type": "keyword",
              "ignore_above": 256
            },
            "chat_room_url_revshare": {
              "type": "keyword",
              "ignore_above": 256
            },
            "current_show": {
              "type": "keyword",
              "ignore_above": 256
            },
            "display_name": {
              "type": "keyword",
              "ignore_above": 256
            },
            "gender": {
              "type": "keyword",
              "ignore_above": 256
            },
            "iframe_embed": {
              "type": "keyword",
              "ignore_above": 256
            },
            "image_url": {
              "type": "keyword",
              "ignore_above": 256
            },
            "image_url_360x270": {
              "type": "keyword",
              "ignore_above": 256
            },
            "is_new": {
              "type": "boolean"
            },
            "location": {
              "type": "keyword",
              "ignore_above": 256
            },
            "num_followers": {
              "type": "long"
            },
            "num_users": {
              "type": "long"
            },
            "recorded": {
              "type": "keyword",
              "ignore_above": 256
            },
            "room_subject": {
              "type": "keyword",
              "ignore_above": 256
            },
            "seconds_online": {
              "type": "long"
            },
            "spoken_languages": {
              "type": "keyword",
              "ignore_above": 256
            },
            "tags": {
              "type": "keyword",
              "ignore_above": 256
            },
            "username": {
              "type": "keyword",
              "ignore_above": 256
            }
          }
        },
        "time": {
          "type": "date"
        }
      }
    }
  }
}`

func createOnlineRoomIndex(client *elastic.Client) {
	// Use the IndexExists service to check if a specified index exists.
	exists, err := client.IndexExists("rooms").Do(context.Background())
	if err != nil {
		// Handle error
		panic(err)
	}
	if !exists {
		// Create a new index.
		createIndex, err := client.CreateIndex("rooms").BodyString(mapping).Do(context.Background())
		if err != nil {
			// Handle error
			panic(err)
		}
		if !createIndex.Acknowledged {
			// Not acknowledged
		}
	}
}

type elasticOM struct {
	Time  time.Time   `json:"time"`
	Model OnlineModel `json:"model"`
}
