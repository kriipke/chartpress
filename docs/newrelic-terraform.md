#  NEW RELIC

provider "newrelic" {
      api_key = "your_new_relic_api_key"
      region  = "US"
    }
    
    resource "newrelic_alert_policy" "microservices_policy" {
      name                = "Microservices Monitoring Policy"
      incident_preference = "PER_POLICY"
    }
    
    # 1. Error Rate Alert
    resource "newrelic_nrql_alert_condition" "high_error_rate" {
      policy_id = newrelic_alert_policy.microservices_policy.id
      name      = "High Error Rate"
      nrql {
        query       = "SELECT percentage(count(*), WHERE error IS true) FROM Transaction SINCE 5 minutes ago"
        evaluation_offset = 3
      }
      critical {
        operator              = "above"
        threshold             = 5
        threshold_duration    = 120
        threshold_occurrences = "ALL"
      }
    }
    
    # 2. High Response Time Alert
    resource "newrelic_nrql_alert_condition" "high_response_time" {
      policy_id = newrelic_alert_policy.microservices_policy.id
      name      = "High Response Time"
      nrql {
        query       = "SELECT average(duration) FROM Transaction SINCE 5 minutes ago"
        evaluation_offset = 3
      }
      critical {
        operator              = "above"
        threshold             = 2
        threshold_duration    = 120
        threshold_occurrences = "ALL"
      }
    }
    
    # 3. CPU Usage Alert
    resource "newrelic_infra_alert_condition" "high_cpu_usage" {
      policy_id = newrelic_alert_policy.microservices_policy.id
      name      = "High CPU Usage"
      type      = "infra_metric"
      comparison = "above"
      threshold  = 80
      duration   = 5
      critical   = true
      metric     = "cpuPercent"
    }
    
    # 4. Memory Usage Alert
    resource "newrelic_infra_alert_condition" "high_memory_usage" {
      policy_id = newrelic_alert_policy.microservices_policy.id
      name      = "High Memory Usage"
      type      = "infra_metric"
      comparison = "above"
      threshold  = 80
      duration   = 5
      critical   = true
      metric     = "memoryPercent"
    }
    
    # 5. Disk Space Usage Alert
    resource "newrelic_infra_alert_condition" "high_disk_space_usage" {
      policy_id = newrelic_alert_policy.microservices_policy.id
      name      = "High Disk Space Usage"
      type      = "infra_metric"
      comparison = "above"
      threshold  = 90
      duration   = 5
      critical   = true
      metric     = "diskSpacePercent"
    }
    
    # 6. Low Throughput Alert
    resource "newrelic_nrql_alert_condition" "low_throughput" {
      policy_id = newrelic_alert_policy.microservices_policy.id
      name      = "Low Throughput"
      nrql {
        query       = "SELECT rate(count(*), 1 minute) FROM Transaction SINCE 5 minutes ago"
        evaluation_offset = 3
      }
      critical {
        operator              = "below"
        threshold             = 10
        threshold_duration    = 300
        threshold_occurrences = "ALL"
      }
    }
    
    # 7. Database Query Time Alert
    resource "newrelic_nrql_alert_condition" "slow_db_query" {
      policy_id = newrelic_alert_policy.microservices_policy.id
      name      = "Slow Database Query"
      nrql {
        query       = "SELECT average(databaseDuration) FROM Transaction SINCE 5 minutes ago"
        evaluation_offset = 3
      }
      critical {
        operator              = "above"
        threshold             = 1
        threshold_duration    = 120
        threshold_occurrences = "ALL"
      }
    }
    
    # 8. External Service Call Duration Alert
    resource "newrelic_nrql_alert_condition" "external_service_latency" {
      policy_id = newrelic_alert_policy.microservices_policy.id
      name      = "High External Service Call Duration"
      nrql {
        query       = "SELECT average(externalDuration) FROM Transaction SINCE 5 minutes ago"
        evaluation_offset = 3
      }
      critical {
        operator              = "above"
        threshold             = 1.5
        threshold_duration    = 120
        threshold_occurrences = "ALL"
      }
    }
    
    # 9. Error Count Alert
    resource "newrelic_nrql_alert_condition" "high_error_count" {
      policy_id = newrelic_alert_policy.microservices_policy.id
      name      = "High Error Count"
      nrql {
        query       = "SELECT count(*) FROM Transaction WHERE error IS true SINCE 5 minutes ago"
        evaluation_offset = 3
      }
      critical {
        operator              = "above"
        threshold             = 100
        threshold_duration    = 120
        threshold_occurrences = "ALL"
      }
    }
    
    # 10. Deployment Alert
    resource "newrelic_nrql_alert_condition" "deployment_spike" {
      policy_id = newrelic_alert_policy.microservices_policy.id
      name      = "Deployment Spike"
      nrql {
        query       = "SELECT count(*) FROM Transaction WHERE deploymentId IS NOT NULL SINCE 5 minutes ago"
        evaluation_offset = 3
      }
      critical {
        operator              = "above"
        threshold             = 10
        threshold_duration    = 120
        threshold_occurrences = "ALL"
      }
    }
