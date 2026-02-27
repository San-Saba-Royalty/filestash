variable "resource_group_name" {
  description = "Azure resource group containing the AKS cluster"
  type        = string
  default     = "sansaba-rg"
}

variable "cluster_name" {
  description = "AKS cluster name"
  type        = string
  default     = "ssr"
}

variable "filestash_image" {
  description = "Full image reference for the filestash container (e.g. sansabaacr.azurecr.io/filestash:abc123)"
  type        = string
}

variable "application_url" {
  description = "Public HTTPS URL filestash is served from"
  type        = string
  default     = "https://files-ssr.prometheusags.ai"
}

variable "azure_storage_account_name" {
  description = "Azure Storage account name for the File Share backend"
  type        = string
  default     = "sansaba"
}

variable "azure_storage_account_key" {
  description = "Azure Storage account key (sensitive)"
  type        = string
  sensitive   = true
}

variable "filestash_admin_password" {
  description = "Admin password for the Filestash admin panel"
  type        = string
  sensitive   = true
}

variable "filestash_fqdn" {
  description = "Public hostname for Filestash (used for the Ingress TLS host and rule)"
  type        = string
  default     = "files-ssr.prometheusags.ai"
}

variable "filestash_shares" {
  description = "Comma-separated Azure File Share names to auto-configure as connections (empty = one connection with access to all shares)"
  type        = string
  default     = ""
}
