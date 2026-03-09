# Generating S3 Dataset Credentials
In this guide, we will walk through the process of creating temporary credentials for accessing Amazon S3 datasets using Amazon Web Services (AWS) Security Token Service (STS). Temporary credentials, often referred to as "scoped-down" or "restricted" credentials, allow you to grant limited access to specific AWS services or resources, such as S3 buckets, enhancing security and access control.

## Prerequisites:
Before we proceed with obtaining temporary credentials for Amazon S3 datasets using AWS Security Token Service (STS), let's cover the essential prerequisites. You will need:

##### AWS Account:
- If you don't already have an AWS account, you'll need to create one. Visit the AWS website and sign up for an AWS account.

##### Steps to Obtain Account ID, Secret Key, and Access Key ID:
- Log into the AWS Management Console:
- Access Your Account ID:
- Generate Secret Key and Access Key ID:

#### Steps to Create an IAM Role with S3 Bucket Access Policy:
- Navigate to IAM:
- Create an IAM Role:
- Attach an IAM policy that defines the permissions you want to grant for S3 access.
- Once the role is created, you can use AWS Security Token Service (STS) to assume the role and obtain temporary credentials with the desired permissions.

Let's proceed and generate temporary credentials by running the service.
- Run Credenrial service using the following command.
```sh
AWS_ACCOUNT_ID=<account-id> AWS_S3_ROLE=<role-name> AWS_SESSION_ROLE=<session-role> AWS_ACCESS_KEY_ID=<access-key-id> AWS_SECRET_ACCESS_KEY=<secret-access-key> ENV=staging go run cmd/credential-manager/main.go
```
- Run Access Manager Service using the following command.
```sh
ENV=staging CREDENTIAL_MANAGER_DNS=<credential-manager-host-address> go run cmd/access-manager/main.go
```
After running the service, you can request for credential using this endpoint `accessmanager.AccessManager.GetDatasetCredential`
```sh
request payload:
{
  "path": "am://data/hpe/bu1/hot-data",
  "operations": [
    0,1
  ],
  "caller_id": "am://user/hpe/bu1/alice"
}
```

The `path` contains the dataset key for which credentials have been requested.
```sh
dataset:
    {
      "Key": "am://data/hpe/bu1/hot-data",
      "Version": 32,
      "Global": 775,
      "StartMillis": 0,
      "EndMillis": 0,
      "Url": "s3://ham-test-1/A",
      "UrlWildcard": "s3://ham-test-2/X/*"
    },
```

The generated credentials will only have access to the operations that they requested on given `url` and `UrlWildcard` paths of the s3 bucket.

## Scope of Temporary Credentials:
According to the above scenario the temporary credentials will have only limited access to the given url` and `UrlWildcard` and custom policy, while assuming role using AWS STS to generate temporary credentials.

##### Credential Scope with requested operations:
Credential will have following operation access to perform certain actions on s3 bucket datasets.
```sh
    Admin: {"s3:ListBucket", "s3:GetObject", "s3:PutObject", "s3:DeleteObject"},
	Write: {"s3:ListBucket", "s3:GetObject", "s3:PutObject"},
	Read:  {"s3:ListBucket", "s3:GetObject"},
	View:  {"s3:ListBucket"},
```

For example:
We have two buckets and there subfolders and objects:
- ham-test-1/
    - A/
    - B/
    - C/
- ham-test-2/
    - X/
    - Y/
    - Z/


If credentials are created with `Admin` for `"Url": "s3://ham-test-1/A"` and  `"UrlWildcard": "s3://ham-test-2/X/*"`, then the scope of credential will only be limited to these url paths and only all operations like view, read, write, delete can be performed. Similar for other operations the access is limited according to the requested permission.
With generated credentials user will not be able to access path like `s3://ham-test-1/B` or any other existing object, the Access Denied error will be thrown if user try to access any other object or folder.

Also if url contains `*` in the data path like this `"UrlWildcard": "s3://ham-test-2/X/*"`, in that case, the credentials scope will automaticaly inherited towards all subfolder and objects with in the `X/` folder.
But if url is like this `"Url": "s3://ham-test-1/A"`, in that case the credential scope will be limited at folder or object level, and will not be inherited if there are any subfolders and objects.

The default credential lifespan is set to 30 minutes. After this period, the temporary credential session expires, rendering the credentials no longer usable.

