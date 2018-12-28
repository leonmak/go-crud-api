# Go CRUD 

Simple REST API for Golang.

## Install dependencies
```
dep ensure
```


## Database config
- Edit & Copy `example-config.json` to `config` folder, renaming the file to `dev.json` 

### Development
- Local postgres instance `dev.json`
    ```json
    { 
      "dbHostName": "localhost",
      "dbPortName": "5432",
      "dbSourceName": "groupbuy"
    }
    ```
- Cloud SQL using proxy:
    - Download proxy: `curl -o cloud_sql_proxy https://dl.google.com/cloudsql/cloud_sql_proxy.darwin.amd64` 
    - Run proxy: `./cloud_sql_proxy -instances="groupbuy-api-2018:asia-southeast1:groupbuy-api-staging"=tcp:5432`
    - Setup database as below and fill in user & password info in `dev-cloudql.json`
    ```json
        { 
          "dbHostName": "localhost",
          "dbPortName": "5432",
          "dbSourceName": "groupbuy",
          "dbUsername": "edit from settings",
          "dbPassword": "edit from settings"
        }
    ```
    - Change env variable `export ENV=dev-cloudql`

### Staging
- In `config/staging.json`, `"dbHostName":"/cloudsql/dealbasin-api-staging:asia-east2:dealbasin-api-staging-pg96"`
- In `app.yaml`
    ```yaml
    env_variables:
      ENV: staging
    beta_settings:
      cloud_sql_instances: /cloudsql/dealbasin-api-staging:asia-east2:dealbasin-api-staging-pg96
    ```

## Database setup

### Local
- Create database: `createdb -h localhost -p 5432 -U postgres groupbuy`
- Create tables: `psql -h localhost -d groupbuy -p 5432 -f ./sql/common/file.sql`
    ```bash
    for i in `ls sql/common`; do
        psql -h localhost -d groupbuy -p 5432 -f ./sql/common/$i
    done
    for i in `ls sql/data`; do
      psql -h localhost -d groupbuy -p 5432 -f ./sql/data/$i
    done
    ```
   
### Cloud SQL
```bash
# - Create database: 
ENV=staging
INSTANCE_NAME=dealbasin-api-${ENV}-pg96
BUCKET_NAME=dealbasin-api-${ENV}-bucket
DATABASE_NAME=dealbasin
REGION=asia-east2
gcloud sql instances create ${INSTANCE_NAME} --region=${REGION} --database-version=POSTGRES_9_6 --tier=db-f1-micro
gcloud sql databases create ${DATABASE_NAME} --instance ${INSTANCE_NAME}

# Edit "dbUsername", "dbPassword" in `config/${ENV}.json`
# gcloud sql users create user123 --instance=${INSTANCE_NAME} --password=pw123
 
# - Create GCS bucket to store sql files:
gsutil -m rm -r gs://${BUCKET_NAME}    
gsutil mb gs://${BUCKET_NAME}    
gsutil -m cp -r ./sql/ gs://${BUCKET_NAME}/
SVC_ACCOUNT_ADDRESS=`gcloud sql instances describe ${INSTANCE_NAME} | grep service | sed -e 's/.*: //'`
gsutil acl ch -u ${SVC_ACCOUNT_ADDRESS}:W gs://${BUCKET_NAME}
gsutil -m acl ch -r -u ${SVC_ACCOUNT_ADDRESS}:R gs://${BUCKET_NAME}/

# - Create tables with sql files:
for i in `ls sql/common`; do
    gcloud sql import sql ${INSTANCE_NAME} gs://${BUCKET_NAME}/sql/common/${i} --database=${DATABASE_NAME}
done
```

### Setup firebase
- Enable email, fb, google sign in
- Create a database if not already created
- Download Service Account Key json under `Settings` > `Service Accounts` and rename to `[ENV]-serviceAccountKey.json`
- Edit Rules, see `firebase-database-rules.md`

### Deploy to App Engine
```bash
gcloud app deploy app-prod.yaml
gcloud app deploy app-staging.yaml
```
