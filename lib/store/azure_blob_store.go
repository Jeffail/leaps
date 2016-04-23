// +build AZURE

package store

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	azure "github.com/azure/azure-sdk-for-go/storage"
	"github.com/cenkalti/backoff"
)

//--------------------------------------------------------------------------------------------------

// AzureConfig - Azure Blob Storage configuration.
type AzureConfig struct {
	Account    string `json:"account" yaml:"account"`
	Secret     string `json:"secret" yaml:"secret"`
	Container  string `json:"container" yaml:"container"`
	AccessType string `json:"access_type" yaml:"access_type"`
}

// NewAzureConfig - Returns a default Azure Blob Storage configuration.
func NewAzureConfig() AzureConfig {
	return AzureConfig{}
}

/*--------------------------------------------------------------------------------------------------
 */

// AzureBlob - Contains configuration and logic for CRUD operations on Azure.
type AzureBlob struct {
	config      AzureConfig
	blobStorage azure.BlobStorageClient
}

// NewAzureBlob - Create a new AzureBlob crud type.
func NewAzureBlob(config AzureConfig) (Type, error) {
	client, err := azure.NewClient(
		config.Account,
		config.Secret,
		azure.DefaultBaseURL,
		azure.DefaultAPIVersion,
		true)
	if err != nil {
		return nil, err
	}
	blobStorage := client.GetBlobService()
	// Ensure the container exists
	var accessType azure.ContainerAccessType
	switch config.AccessType {
	case "":
		accessType = azure.ContainerAccessTypePrivate
	case "blob":
		accessType = azure.ContainerAccessTypeBlob
	case "container":
		accessType = azure.ContainerAccessTypeContainer
	default:
		err := fmt.Errorf(
			"azure container access_type: '%s' is invalid",
			config.AccessType)
		return nil, err
	}
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = 500 * time.Second
	b.RandomizationFactor = 0.5
	b.Multiplier = 1.5
	b.MaxInterval = 20
	b.MaxElapsedTime = 1 * time.Minute
	err = backoff.Retry(func() error {
		_, err := blobStorage.CreateContainerIfNotExists(config.Container, accessType)
		return err
	}, b)
	if err != nil {
		return nil, err
	}
	// Return AzureBlob
	return &AzureBlob{
		config:      config,
		blobStorage: blobStorage,
	}, nil
}

// Create - Create a new document in azure blob storage
func (m *AzureBlob) Create(doc Document) error {
	return m.Update(doc)
}

// Update - Update document in azure blob storage
func (m *AzureBlob) Update(doc Document) error {
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = 500 * time.Second
	b.RandomizationFactor = 0.5
	b.Multiplier = 2
	b.MaxInterval = 60
	b.MaxElapsedTime = 15 * time.Minute
	return backoff.Retry(func() error {
		r := strings.NewReader(doc.Content)
		return m.blobStorage.CreateBlockBlobFromReader(
			m.config.Container, doc.ID, uint64(r.Len()), r,
		)
	}, b)
}

// Read - Read document from a azure blob storage
func (m *AzureBlob) Read(id string) (Document, error) {
	doc := Document{
		ID: id,
	}
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = 500 * time.Second
	b.RandomizationFactor = 0.5
	b.Multiplier = 1.5
	b.MaxInterval = 10
	b.MaxElapsedTime = 45 * time.Second
	var retErr error
	err := backoff.Retry(func() error {
		rc, err := m.blobStorage.GetBlob(m.config.Container, id)
		if rc != nil {
			defer rc.Close()
		}
		if err != nil {
			switch e := err.(type) {
			case azure.AzureStorageServiceError:
				if e.StatusCode == 404 {
					retErr = ErrDocumentNotExist
					return nil
				}
				if e.StatusCode >= 500 {
					return fmt.Errorf("Internal server error from azure: %d", e.StatusCode)
				}
				// Don't retry on non-500 errors
				retErr = e
				return nil
			default:
				return err
			}
		}
		// Read body
		b := new(bytes.Buffer)
		_, err = b.ReadFrom(rc)
		if err != nil {
			return err
		}
		doc.Content = b.String()
		return nil
	}, b)
	if retErr != nil {
		return Document{}, retErr
	}
	return doc, err
}
