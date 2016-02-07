// +build !AZURE

package store

/*--------------------------------------------------------------------------------------------------
 */

/*
AzureStorageConfig - Azure Blob Storage configuration.
*/
type AzureStorageConfig struct{}

/*
NewAzureStorageConfig - Returns a default Azure Blob Storage configuration.
*/
func NewAzureStorageConfig() AzureStorageConfig {
	return AzureStorageConfig{}
}

/*--------------------------------------------------------------------------------------------------
 */
