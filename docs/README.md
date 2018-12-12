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
- In `config/staging.json`, `"dbHostName":"/cloudsql/groupbuy-api-2018:asia-southeast1:groupbuy-api-staging"`
- In `app.yaml`
    ```yaml
    beta_settings:
      cloud_sql_instances: groupbuy-api-2018:asia-southeast1:groupbuy-api-staging
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
DATABASE_NAME=groupbuy
INSTANCE_NAME=groupbuy-api-staging
BUCKET_NAME=groupbuy-api
gcloud sql databases create ${DATABASE_NAME} --instance ${INSTANCE_NAME}

# - Create GCS bucket to store sql files:
gsutil mb gs://${BUCKET_NAME}    
gsutil cp -r ./sql/ gs://${BUCKET_NAME}/
SVC_ACCOUNT_ADDRESS=`gcloud sql instances describe groupbuy-api-staging | grep service | sed -e 's/.*: //'`
gsutil acl ch -u ${SVC_ACCOUNT_ADDRESS}:W gs://${BUCKET_NAME}
gsutil acl ch -r -u ${SVC_ACCOUNT_ADDRESS}:R gs://${BUCKET_NAME}/

# - Create tables with sql files:
for i in `ls sql/common`; do
    gcloud sql import sql ${INSTANCE_NAME} gs://${BUCKET_NAME}/sql/common/${i} --database=${DATABASE_NAME}
done
```


## Routes:
- `/deals`
    - GET
        - `search_text`
            - `string`, fuzzy text search on `title` column
        - `city_id`
            - `serial`, reference cities(id)
        - `poster_id` (optional)
            - `string`
        - `category_id` (optional)
            - `serial`
        - `after`, `before` (optional)
            - `string` iso8601 format, e.g. `2011-10-05T14:48:00.000Z`
            - for paginating - after most recent, and before least recent item 
            - must use both `after` & `before` if used  
        - `show_inactive` (optional)
            - `bool`, e.g. `true`(show all) / `false` (default, show `inactive_at` is null)
        - `lat`, `lng` (optional)
            - `float64`, up to 64 digits, e.g. `1.3521`
            - requires `radius_km`
        - `radius_km` (optional)
            - `int`
            - requires `lat` & `lng`
- `/deal`
    - POST
        - accepts JSON payload with keys: 
        `title`,`description`,`thumbnailId`,`latitude`,`longitude`,`locationText`,
        `expectedPrice`,`categoryId`,`posterId`,`cityId`
        - returns uuid string on successful
- `/deal/{id}`
    - GET
    - PUT
    - DELETE

- `/deal/{id}/members`
- `/deal/{id}/comments`
- `/deal/{id}/images`
- `/categories/` 
- `/user/{id}`