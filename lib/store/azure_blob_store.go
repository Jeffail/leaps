package store

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	azure "github.com/azure/azure-sdk-for-go/storage"
	"github.com/cenkalti/backoff"
)

/*
Azure Blob Storage configuration
*/
type AzureStorageConfig struct {
	Account    string `json:"account" yaml:"account"`
	Secret     string `json:"secret" yaml:"secret"`
	Container  string `json:"container" yaml:"container"`
	AccessType string `json:"access_type" yaml:"access_type"`
}

type AzureBlobStore struct {
	config      AzureStorageConfig
	blobStorage azure.BlobStorageClient
}

func GetAzureBlobStore(config Config) (*AzureBlobStore, error) {
	client, err := azure.NewClient(
		config.AzureBlobStore.Account,
		config.AzureBlobStore.Secret,
		azure.DefaultBaseURL,
		azure.DefaultAPIVersion,
		true)
	if err != nil {
		return nil, err
	}
	blobStorage := client.GetBlobService()
	// Ensure the container exists
	var accessType azure.ContainerAccessType
	switch config.AzureBlobStore.AccessType {
	case "":
		accessType = azure.ContainerAccessTypePrivate
	case "blob":
		accessType = azure.ContainerAccessTypeBlob
	case "container":
		accessType = azure.ContainerAccessTypeContainer
	default:
		err := fmt.Errorf(
			"azure container access_type: '%s' is invalid",
			config.AzureBlobStore.AccessType)
		return nil, err
	}
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = 500 * time.Second
	b.RandomizationFactor = 0.5
	b.Multiplier = 1.5
	b.MaxInterval = 20
	b.MaxElapsedTime = 1 * time.Minute
	err = backoff.Retry(func() error {
		_, err := blobStorage.CreateContainerIfNotExists(config.AzureBlobStore.Container, accessType)
		return err
	}, b)
	if err != nil {
		return nil, err
	}
	// Return AzureBlobStore
	return &AzureBlobStore{
		config:      config.AzureBlobStore,
		blobStorage: blobStorage,
	}, nil
}

/*
Create - Create a new document in azure blob storage
*/
func (m *AzureBlobStore) Create(id string, doc Document) error {
	return m.Store(id, doc)
}

/*
Store - Store document in azure blob storage
*/
func (m *AzureBlobStore) Store(id string, doc Document) error {
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = 500 * time.Second
	b.RandomizationFactor = 0.5
	b.Multiplier = 2
	b.MaxInterval = 60
	b.MaxElapsedTime = 15 * time.Minute
	return backoff.Retry(func() error {
		r := strings.NewReader(doc.Content)
		return m.blobStorage.CreateBlockBlobFromReader(m.config.Container, id, uint64(r.Len()), r)
	}, b)
}

/*
Fetch - Fetch document from a azure blob storage
*/
func (m *AzureBlobStore) Fetch(id string) (Document, error) {
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
