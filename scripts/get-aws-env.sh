#!/bin/bash

set +e
which aws >/dev/null 2>&1
if [[ $? -ne 0 ]]; then
  echo -e "\nERROR! The AWS CLI is required to run the AWS Cognito configuration commands."
  exit 1
fi
which jq >/dev/null 2>&1
if [[ $? -ne 0 ]]; then
  echo -e "\nERROR! The JQ CLI is required for AWS API response processing."
  exit 1
fi

set -euo pipefail

scripts_dir=$(dirname $BASH_SOURCE)
home_dir=$(cd ${scripts_dir}/.. && pwd)

function usage() {
  echo -e "\nUSAGE: configure.sh [options]\n"
  echo -e "  -e|--env [ENV]  the deployment environment"
  echo -e "  -d|--debug      enable trace output"
  echo -e "  -h|--help       show this help"
}

env=dev
while [[ $# -gt 0 ]]; do
  case "$1" in
    -e|--env)
      env=$2
      shift
      ;;
    -d|--debug)
      set -x
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo -e "\nERROR! Unknown option \"$1\"."
      usage
      exit 1
      ;;
  esac
  shift
done

env_name="mycs${env}"
aws_region=${AWS_DEFAULT_REGION:-us-east-1}

identity_outputs=$(aws --region ${aws_region} \
  cloudformation describe-stacks \
  --stack-name ${env_name}-identity \
  | jq '.Stacks[0].Outputs[]' \
  | sed 's|\\n|\\\\n|g')

api_outputs=$(aws --region ${aws_region} \
  cloudformation describe-stacks \
  --stack-name ${env_name}-api \
  | jq '.Stacks[0].Outputs[]' \
  | sed 's|\\n|\\\\n|g')

# Update Hosted UI for CLI client
cognitoRegion=$(echo "$identity_outputs" | jq -r 'select(.OutputKey=="Region") | .OutputValue')
userPoolId=$(echo "$identity_outputs" | jq -r 'select(.OutputKey=="UserPoolId") | .OutputValue')
cliClientID=$(echo "$identity_outputs" | jq -r 'select(.OutputKey=="UserPoolCLIClientId") | .OutputValue')
cliClientSecret=$(echo "$identity_outputs" | jq -r 'select(.OutputKey=="UserPoolCLIClientSecret") | .OutputValue')
appsyncRegion=$(echo "$api_outputs" | jq -r 'select(.OutputKey=="Region") | .OutputValue')
userSpaceApiUrl=$(echo "$api_outputs" | jq -r 'select(.OutputKey=="UserSpaceApiUrl") | .OutputValue')

# aws amplify config for mycloudspace web app
cat << ---EOF > ${home_dir}/config/aws.go
package config

// **** DO NOT EDIT ****
// 
// This file is auto-generated by scripts/get-aws-env.sh.

const AWS_COGNITO_REGION = "${cognitoRegion}"
const AWS_COGNITO_USER_POOL_ID = "${userPoolId}"

const CLIENT_ID = "${cliClientID}"
const CLIENT_SECRET = "${cliClientSecret}"
const AUTH_URL = "https://${env_name}.auth.us-east-1.amazoncognito.com/login"
const TOKEN_URL = "https://${env_name}.auth.us-east-1.amazoncognito.com/oauth2/token"
const USER_INFO_URL = "https://${env_name}.auth.us-east-1.amazoncognito.com/oauth2/userInfo"

const AWS_APPSYNC_REGION = "${appsyncRegion}"
const AWS_USERSPACE_API_URL = "${userSpaceApiUrl}"
---EOF