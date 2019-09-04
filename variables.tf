variable "dd_api_key" {
  description = "Datadog API Key"
}

variable "dd_app_key" {
  description = "Datadog APP Key"
}

variable "aws_account_id" {
  description = "AWS account ID"
}

variable "slack_verification_token" {
  description = "Slack verification token"
}

variable "aws_endpoint_region" {
  description = "AWS reqion the endpoint will be deployed"
}

variable "datadog_team" {
  description = "Datadog team name used to build the dash board URL"
}
