# alive-arns

`alive-arns` print AWS Resource Names across all regions.

## Usage

``` console
$ alive-arns
2022-05-02T12:08:56+09:00 INF Checking eu-north-1
2022-05-02T12:09:03+09:00 INF Checking ap-south-1
2022-05-02T12:09:04+09:00 INF Checking eu-west-3
2022-05-02T12:09:05+09:00 INF Checking eu-west-2
2022-05-02T12:09:06+09:00 INF Checking eu-west-1
2022-05-02T12:09:08+09:00 INF Checking ap-northeast-3
2022-05-02T12:09:08+09:00 INF Checking ap-northeast-2
2022-05-02T12:09:08+09:00 INF Checking ap-northeast-1
2022-05-02T12:09:09+09:00 INF Checking sa-east-1
2022-05-02T12:09:11+09:00 INF Checking ca-central-1
2022-05-02T12:09:12+09:00 INF Checking ap-southeast-1
2022-05-02T12:09:12+09:00 INF Checking ap-southeast-2
2022-05-02T12:09:13+09:00 INF Checking eu-central-1
2022-05-02T12:09:14+09:00 INF Checking us-east-1
2022-05-02T12:09:16+09:00 INF Checking us-east-2
2022-05-02T12:09:16+09:00 INF Checking us-west-1
2022-05-02T12:09:17+09:00 INF Checking us-west-2
arn:aws:acm:us-east-1:01234567890:certificate/xxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxx
arn:aws:apigateway:ap-northeast-1::/restapis/xxxxxxxxxx
arn:aws:apigateway:ap-northeast-1::/restapis/yyyyyyyyyy
arn:aws:backup:ap-northeast-1:01234567890:backup-vault:xxxxxxx
arn:aws:backup:ap-northeast-2:01234567890:backup-vault:yyyyyyy
arn:aws:backup:ap-northeast-3:01234567890:backup-vault:zzzzzzz
arn:aws:backup:ap-northeast-1:01234567890:recovery-point:xxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxx
arn:aws:backup:ap-northeast-1:01234567890:recovery-point:xxxxxxx-xxxx-xxxx-xxxx-yyyyyyyyyyy
[...]
arn:aws:iam::aws:policy/AWSSupportAccess
arn:aws:iam::aws:policy/AWSThinkboxAWSPortalAdminPolicy
arn:aws:iam::aws:policy/AWSThinkboxAWSPortalGatewayPolicy
arn:aws:iam::aws:policy/AWSThinkboxAWSPortalWorkerPolicy
arn:aws:iam::aws:policy/AWSThinkboxAssetServerPolicy
[...]
arn:aws:route53:::healthcheck/xxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxx
arn:aws:route53:::healthcheck/xxxxxxx-xxxx-xxxx-xxxx-yyyyyyyyyyy
arn:aws:route53:::healthcheck/xxxxxxx-xxxx-xxxx-xxxx-zzzzzzzzzzz
arn:aws:route53:::healthcheck/xxxxxxx-xxxx-xxxx-xxxx-aaaaaaaaaaa
arn:aws:route53:::healthcheck/xxxxxxx-xxxx-xxxx-xxxx-bbbbbbbbbbb
```

### Using jq

#### Count resources by region

``` console
$ alive-arns -t json | jq 'group_by(.region) | map({"region": .[0].region, "count": length})'
[
  {
    "region": "",
    "count": 3345
  },
  {
    "region": "ap-northeast-1",
    "count": 438
  },
  {
    "region": "us-east-1",
    "count": 124
  }
]
```

## Supported ARNs

- [ARNs that can be retrieved with the Resource Groups Tagging API](https://docs.aws.amazon.com/resourcegroupstagging/latest/APIReference/supported-services.html)
- `arn:aws:iam:*`
