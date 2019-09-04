output "opslog_url" {
  value       = "${aws_api_gateway_rest_api.opslog_API.id}"
  description = "This is the URL base to give to slack for /opslog and /opsloghelp"
}