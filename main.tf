//
// AWS API Gateway
//

resource "aws_api_gateway_rest_api" "opslog_API" {
  name        = "opslogAPI"
  description = "API for slack opslog commands"
}

resource "aws_api_gateway_resource" "opslog_resource" {
  rest_api_id = "${aws_api_gateway_rest_api.opslog_API.id}"
  parent_id   = "${aws_api_gateway_rest_api.opslog_API.root_resource_id}"
  path_part   = "opslog"
}

resource "aws_api_gateway_method" "opslog_method" {
  rest_api_id   = "${aws_api_gateway_rest_api.opslog_API.id}"
  resource_id   = "${aws_api_gateway_resource.opslog_resource.id}"
  http_method   = "POST"
  authorization = "NONE"
}

resource "aws_api_gateway_integration" "opslog_api_gw_integration" {
  rest_api_id             = "${aws_api_gateway_rest_api.opslog_API.id}"
  resource_id             = "${aws_api_gateway_resource.opslog_resource.id}"
  http_method             = "${aws_api_gateway_method.opslog_method.http_method}"
  integration_http_method = "POST"
  type                    = "AWS_PROXY"
  uri                     = "arn:aws:apigateway:${var.aws_region}:lambda:path/2015-03-31/functions/${aws_lambda_function.opslog_lambda.arn}/invocations"
}

//
// AWS Lambda
//

resource "aws_lambda_function" "opslog_lambda" {
  filename         = "${path.module}/package/opslog.zip"
  function_name    = "opslog"
  role             = "${aws_iam_role.oplog_lambda_IAM_role.arn}"
  handler          = "opslog"
  source_code_hash = "${filebase64sha256("${path.module}/package/opslog.zip")}"
  runtime          = "go1.x"
  environment {
    variables = {
      SLACK_VERIFICATION_TOKEN = "${var.slack_verification_token}",
      SLACK_OAUTH_TOKEN        = "${var.slack_oauth_token}",
      DB_TABLE_NAME            = "${aws_dynamodb_table.opslog_table.id}"
    }
  }
}

resource "aws_lambda_permission" "api_gw_lambda_perms" {
  statement_id  = "AllowExecutionFromAPIGateway"
  action        = "lambda:InvokeFunction"
  function_name = "${aws_lambda_function.opslog_lambda.function_name}"
  principal     = "apigateway.amazonaws.com"
  # More: http://docs.aws.amazon.com/apigateway/latest/developerguide/api-gateway-control-access-using-iam-policies-to-invoke-api.html
  source_arn = "arn:aws:execute-api:${var.aws_region}:${var.aws_account_id}:${aws_api_gateway_rest_api.opslog_API.id}/*/${aws_api_gateway_method.opslog_method.http_method}${aws_api_gateway_resource.opslog_resource.path}"
}

resource "aws_iam_role" "oplog_lambda_IAM_role" {
  name = "oplog_lambda_IAM_role"

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "lambda.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}

//
// AWS Cloudwatch
//

resource "aws_cloudwatch_log_group" "opslog_logging_group" {
  name = "/aws/lambda/opslog"
  retention_in_days = 120
}

resource "aws_iam_role_policy" "opslog_logging_policy" {
  name = "opslog_logging_policy"
  role = "${aws_iam_role.oplog_lambda_IAM_role.id}"

  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
        {
      "Action": [
        "logs:CreateLogGroup",
        "logs:CreateLogStream",
        "logs:PutLogEvents"
      ],
      "Resource": "arn:aws:logs:*:*:*",
      "Effect": "Allow"
    }
  ]
}
EOF
}

//
// AWS DynamoDB
//

resource "aws_dynamodb_table" "opslog_table" {
  name      = "opslog"
  hash_key  = "Channel"
  range_key = "DateTime"

  billing_mode = "PAY_PER_REQUEST"

  attribute {
    name = "Channel"
    type = "S"
  }

  attribute {
    name = "User"
    type = "S"
  }

  attribute {
    name = "DateTime"
    type = "S"
  }

  global_secondary_index {
    name            = "UserIndex"
    hash_key        = "User"
    range_key       = "DateTime"
    projection_type = "ALL"
  }

  tags = {
    App = "opslog"
  }
}

resource "aws_iam_role_policy" "opslog_dynamodb_policy" {
  name = "opslog_dynamodb_policy"
  role = "${aws_iam_role.oplog_lambda_IAM_role.id}"

  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
        {
      "Action": [
        "dynamodb:DeleteItem",
        "dynamodb:GetItem",
        "dynamodb:PutItem",
        "dynamodb:Query",
        "dynamodb:UpdateItem",
        "dynamodb:BatchGetItem",
        "dynamodb:Scan"
      ],
      "Resource": "arn:aws:dynamodb:*:*:table/${aws_dynamodb_table.opslog_table.name}*",
      "Effect": "Allow"
    }
  ]
}
EOF
}