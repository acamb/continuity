package client

import (
	"bytes"
	"continuity/client/config"
	"continuity/common/requests"
	"continuity/common/responses"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const POOL_ENDPOINT = "/pools"

type Client struct {
	httpclient    http.Client
	configuration *config.Configuration
	endpoint      string
}

func NewClient(configuration *config.Configuration) Client {
	return Client{
		httpclient:    http.Client{},
		configuration: configuration,
		endpoint:      configuration.Host + ":" + fmt.Sprint(configuration.Port) + POOL_ENDPOINT,
	}
}

func handleError(resp *http.Response) {
	readBody, _ := io.ReadAll(resp.Body)
	respError := responses.ErrorResponse{}
	_ = json.Unmarshal(readBody, &respError)
	log.Fatalf("Request failed, server responded: %d - %s", resp.StatusCode, respError.Error)
}

func (c *Client) GetVersion() (string, error) {
	resp, err := c.httpclient.Get(c.configuration.Host + ":" + fmt.Sprint(c.configuration.Port) + "/version")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		handleError(resp)
	} else {
		readBody, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		versionResponse := responses.VersionResponse{}
		err = json.Unmarshal(readBody, &versionResponse)
		if err != nil {
			log.Fatal(err)
		}
		return versionResponse.Version, nil
	}
	return "", nil
}

func (c *Client) AddPool(request requests.CreatePoolRequest) {
	body, err := json.Marshal(request)
	if err != nil {
		log.Fatal(err)
	}
	resp, err := c.httpclient.Post(c.endpoint, "", bytes.NewReader(body))
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		handleError(resp)
	} else {
		log.Printf("Pool added successfully for %s\n", request.Hostname)
	}
}

func (c *Client) RemovePool(hostname string) {
	req, err := http.NewRequest(http.MethodDelete, c.endpoint+"/"+base64.RawURLEncoding.EncodeToString([]byte(hostname)), nil)
	if err != nil {
		log.Fatal(err)
	}
	resp, err := c.httpclient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		handleError(resp)
	} else {
		log.Printf("Pool %s removed successfully\n", hostname)
	}
}

func (c *Client) ListPools() {
	resp, err := c.httpclient.Get(c.endpoint)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		handleError(resp)
	} else {
		readBody, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		poolsResponse := responses.ListPoolResponse{}
		err = json.Unmarshal(readBody, &poolsResponse)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("Configured Pools:")
		for _, pool := range poolsResponse.Pools {
			log.Printf("   - %s\n", pool)
		}
	}
}

func (c *Client) GetPoolConfig(hostname string, printJson bool) {
	resp, err := c.httpclient.Get(c.endpoint + "/" + base64.RawURLEncoding.EncodeToString([]byte(hostname)))
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		handleError(resp)
	} else {
		readBody, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		poolResponse := responses.PoolResponse{}
		err = json.Unmarshal(readBody, &poolResponse)
		if err != nil {
			log.Fatal(err)
		}
		if !printJson {
			log.Printf(poolResponse.String())
		} else {
			jsonOutput, err := json.MarshalIndent(poolResponse, "", "  ")
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(string(jsonOutput))
		}
	}
}

func (c *Client) GetPoolStats(hostname string, printJson bool) {
	resp, err := c.httpclient.Get(c.endpoint + "/" + base64.RawURLEncoding.EncodeToString([]byte(hostname)) + "/stats")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		handleError(resp)
	} else {
		readBody, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		poolStatsResponse := responses.PoolStatsResponse{}
		err = json.Unmarshal(readBody, &poolStatsResponse)
		if err != nil {
			log.Fatal(err)
		}
		if !printJson {
			log.Printf(poolStatsResponse.String())
		} else {
			jsonOutput, err := json.MarshalIndent(poolStatsResponse, "", "  ")
			if err != nil {
				log.Fatal(err)
			}
			log.Println(string(jsonOutput))
		}
	}
}

func (c *Client) UpdatePool(request requests.UpdatePoolRequest) {
	body, err := json.Marshal(request)
	log.Println("Updating pool with request:", string(body))
	if err != nil {
		log.Fatal(err)
	}
	resp, err := c.httpclient.Post(c.endpoint+"/"+base64.RawURLEncoding.EncodeToString([]byte(request.Hostname)), "", bytes.NewReader(body))
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		handleError(resp)
	} else {
		log.Printf("Pool %s updated successfully\n", request.Hostname)
	}
}

func (c *Client) AddServer(pool string, request requests.AddServerRequest) {
	body, err := json.Marshal(request)
	if err != nil {
		log.Fatal(err)
	}
	resp, err := c.httpclient.Post(c.endpoint+"/"+base64.RawURLEncoding.EncodeToString([]byte(pool))+"/server", "", bytes.NewReader(body))
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		handleError(resp)
	} else {
		log.Printf("Server %s added successfully to pool %s\n", request.NewServerAddress, pool)
	}
}

func (c *Client) RemoveServer(pool string, serverId string) {
	req, err := http.NewRequest(http.MethodDelete, c.endpoint+"/"+base64.RawURLEncoding.EncodeToString([]byte(pool))+"/"+serverId, nil)
	if err != nil {
		log.Fatal(err)
	}
	resp, err := c.httpclient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		handleError(resp)
	} else {
		log.Printf("Server %s removed successfully from pool %s\n", serverId, pool)
	}
}

func (c *Client) Transaction(pool string, request requests.TransactionRequest) {
	body, err := json.Marshal(request)
	if err != nil {
		log.Fatal(err)
	}
	resp, err := c.httpclient.Post(c.endpoint+"/"+base64.RawURLEncoding.EncodeToString([]byte(pool))+"/transaction", "", bytes.NewReader(body))
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		handleError(resp)
	} else {
		readBody, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		txResponse := responses.TransactionResponse{}
		err = json.Unmarshal(readBody, &txResponse)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Transaction %s in progress...\n", txResponse.TransactionId)
		counter := 0
		for {
			counter++
			time.Sleep(1 * time.Second)
			resp, err = c.httpclient.Get(c.endpoint + "/transaction/" + txResponse.TransactionId)
			readBody, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Fatal(err)
			}
			txResponse := responses.TransactionResponse{}
			err = json.Unmarshal(readBody, &txResponse)
			if err != nil {
				log.Fatal(err)
			}
			if txResponse.Completed {
				if txResponse.Error != "" {
					log.Printf("Transaction %s completed with error: %s\n", txResponse.TransactionId, txResponse.Error)
				} else {
					log.Printf("Transaction %s completed successfully\n", txResponse.TransactionId)
				}
				break
			}
			if counter%10 == 0 {
				log.Printf("Transaction %s still in progress...\n", txResponse.TransactionId)
			}
		}
	}
}
