# compliance-hub-plugin-jenkins-master

## Environment variables used for reading env vars (secrets) directly from secrets manager.
## Only AWS SM is currently supported and hence the sample values for the env vars shown below are AWS specific.
## Other keys and/or values may be needed for other types of secrets managers
## These env vars MUST be set in the helm chart
AWS_REGION = us-east-1
SECRET_MANAGER = AWS_SM
SECRET_ID = cbc-sbx1a-secrets-scan-manager
