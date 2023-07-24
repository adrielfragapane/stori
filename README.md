# 🚀 Stori Challenge

## Description

The objective of this program is to process the csv files that are uploaded to an S3 bucker and process their information, to then send an email with the summary of account movements.

When a csv file is uploaded to an specific S3 bucket, a lambda function is triggered to read the file and process it.

## Deploy:

1. Generating file to create lamda
  
  * With .zip file: 

      * macOS and Linux:

        ```bash
        GOOS=linux GOARCH=amd64 go build -o stori cmd/stori/main.go
        ```
        
        ```bash
        zip stori.zip stori
        ```

      * Windows:

        ```bash
        go install github.com/aws/aws-lambda-go/cmd/build-lambda-zip@latest
        ```

        ```bash
        set GOOS=linux
        set GOARCH=amd64
        set CGO_ENABLED=0
        go build -o stori cmd/stori/main.go
        %USERPROFILE%\Go\bin\build-lambda-zip.exe -o stori.zip stori
        ```

    * With docker file:

      * Build the image

        ```bash
        docker build -t stori .
        ```
    
      * Push to ECR

        ```bash
        docker push <your-account>
        ```

2. Create aws lambda function with .zip or docker

3. Set handler **main**

4. Define env variables:

    * DB_HOST
    * DB_NAME
    * DB_PASS
    * DB_PORT
    * DB_USER
    * EMAIL_USER
    * EMAIL_PASS
    * EMAIL_TO
    

5. Create Trigger
    * Source: S3
    * Event types: POST - PUT
    * Suffix: .csv

