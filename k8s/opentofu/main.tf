terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 4.0"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.0"
    }
  }
}

# ── Providers ──────────────────────────────────────────────────────────────────

provider "azurerm" {
  features {}
}

provider "kubernetes" {
  # In CI, az aks get-credentials writes ~/.kube/config before tofu runs.
  config_path    = "~/.kube/config"
  config_context = var.cluster_name
}

# ── Config Secret ──────────────────────────────────────────────────────────────
# Filestash reads its runtime config from /app/data/state/config/config.json.
# We inject the Azure File Share credentials here so they never live in source.

locals {
  filestash_config = jsonencode({
    general = {
      admin_password = var.filestash_admin_password
    }
    features   = {}
    log        = {}
    email      = {}
    oauth      = {}
    connections = [
      {
        type         = "azurefileshare"
        label        = "Azure File Share"
        account_name = var.azure_storage_account_name
        account_key  = var.azure_storage_account_key
      }
    ]
  })
}

resource "kubernetes_secret_v1" "filestash_config" {
  metadata {
    name      = "filestash-config"
    namespace = "default"

    labels = {
      "app.kubernetes.io/name"       = "filestash"
      "app.kubernetes.io/managed-by" = "opentofu"
    }
  }

  data = {
    "config.json" = local.filestash_config
  }
}

# ── Persistent storage ─────────────────────────────────────────────────────────
# Stores filestash state: share links, search index, session cache.
# Uses the cluster's default StorageClass (Azure Disk via AKS).

resource "kubernetes_persistent_volume_claim_v1" "filestash_data" {
  metadata {
    name      = "filestash-data"
    namespace = "default"

    labels = {
      "app.kubernetes.io/name"       = "filestash"
      "app.kubernetes.io/managed-by" = "opentofu"
    }
  }

  spec {
    access_modes = ["ReadWriteOnce"]

    resources {
      requests = {
        storage = "2Gi"
      }
    }
  }

  # Prevent accidental deletion of the PVC when running tofu destroy.
  lifecycle {
    prevent_destroy = true
  }
}

# ── Deployment ─────────────────────────────────────────────────────────────────
# Single replica — ReadWriteOnce PVC limits us to one pod on AKS by default.
# Filestash is designed for single-instance deployments.
#
# NAMESPACE: "default" — required because Oathkeeper routes requests to
# http://filestash.default.svc.cluster.local:8334

resource "kubernetes_deployment_v1" "filestash" {
  metadata {
    name      = "filestash"
    namespace = "default"

    labels = {
      "app.kubernetes.io/name"       = "filestash"
      "app.kubernetes.io/managed-by" = "opentofu"
    }
  }

  spec {
    replicas = 1

    selector {
      match_labels = {
        "app.kubernetes.io/name" = "filestash"
      }
    }

    strategy {
      type = "Recreate" # Required with ReadWriteOnce PVC
    }

    template {
      metadata {
        labels = {
          "app.kubernetes.io/name" = "filestash"
        }

        annotations = {
          # Force pod restart when the config secret changes.
          "checksum/config" = sha256(local.filestash_config)
        }
      }

      spec {
        container {
          name  = "filestash"
          image = var.filestash_image

          port {
            name           = "http"
            container_port = 8334
            protocol       = "TCP"
          }

          env {
            name  = "APPLICATION_URL"
            value = var.application_url
          }

          # Mount runtime config from secret — overrides baked-in config.json.
          volume_mount {
            name       = "config"
            mount_path = "/app/data/state/config/config.json"
            sub_path   = "config.json"
            read_only  = true
          }

          # Mount persistent state (shares, cache, search index).
          # Excludes config/ — that directory is handled by the secret mount above.
          volume_mount {
            name       = "data"
            mount_path = "/app/data/state"
          }

          resources {
            requests = {
              cpu    = "100m"
              memory = "128Mi"
            }
            limits = {
              cpu    = "500m"
              memory = "512Mi"
            }
          }

          liveness_probe {
            http_get {
              path = "/api/about"
              port = 8334
            }
            initial_delay_seconds = 15
            period_seconds        = 30
            failure_threshold     = 3
          }

          readiness_probe {
            http_get {
              path = "/api/about"
              port = 8334
            }
            initial_delay_seconds = 10
            period_seconds        = 10
          }
        }

        volume {
          name = "config"

          secret {
            secret_name = kubernetes_secret_v1.filestash_config.metadata[0].name
          }
        }

        volume {
          name = "data"

          persistent_volume_claim {
            claim_name = kubernetes_persistent_volume_claim_v1.filestash_data.metadata[0].name
          }
        }
      }
    }
  }

  depends_on = [
    kubernetes_secret_v1.filestash_config,
    kubernetes_persistent_volume_claim_v1.filestash_data,
  ]
}

# ── Service ────────────────────────────────────────────────────────────────────
# Named "filestash" in the "default" namespace so Oathkeeper can reach it at
# http://filestash.default.svc.cluster.local:8334 without any further config.

# ── Ingress ────────────────────────────────────────────────────────────────────
# Must live in the "iam" namespace so it can reference the oathkeeper-proxy
# Service, which also lives there. cert-manager issues the TLS certificate.
# All traffic goes through Oathkeeper for JWT validation before reaching
# the filestash pod.

locals {
  filestash_ingress_annotations = {
    "cert-manager.io/cluster-issuer"                    = "letsencrypt-prod"
    "nginx.ingress.kubernetes.io/ssl-redirect"          = "true"
    "nginx.ingress.kubernetes.io/force-ssl-redirect"    = "true"
    "nginx.ingress.kubernetes.io/proxy-body-size"       = "8m"
  }
}

resource "kubernetes_ingress_v1" "filestash" {
  metadata {
    name      = "filestash-ingress"
    namespace = "iam"

    annotations = local.filestash_ingress_annotations

    labels = {
      "app.kubernetes.io/name"       = "filestash"
      "app.kubernetes.io/managed-by" = "opentofu"
    }
  }

  spec {
    ingress_class_name = "nginx"

    tls {
      hosts       = [var.filestash_fqdn]
      secret_name = "${replace(var.filestash_fqdn, ".", "-")}-tls"
    }

    rule {
      host = var.filestash_fqdn

      http {
        path {
          path      = "/"
          path_type = "Prefix"

          backend {
            service {
              name = "oathkeeper-proxy"
              port {
                number = 4455
              }
            }
          }
        }
      }
    }
  }
}

# ── Service ────────────────────────────────────────────────────────────────────
resource "kubernetes_service_v1" "filestash" {
  metadata {
    name      = "filestash"
    namespace = "default"

    labels = {
      "app.kubernetes.io/name"       = "filestash"
      "app.kubernetes.io/managed-by" = "opentofu"
    }
  }

  spec {
    selector = {
      "app.kubernetes.io/name" = "filestash"
    }

    port {
      name        = "http"
      port        = 8334
      target_port = 8334
      protocol    = "TCP"
    }

    type = "ClusterIP"
  }
}
