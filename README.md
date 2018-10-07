# update_name

Do update_name via CloudWatch Event.

## deploy

`make deploy profile=[AWS_PROFILE] bucket=[DEPLOY_S3_BUCKET]`

## local invoke

1. cp event.json.sample event.json
2. `make invoke`