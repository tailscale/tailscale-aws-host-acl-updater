package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
)

func getApiKey() (string, error) {
	region := os.Getenv("AWS_REGION")
	secretId := os.Getenv("SECRET_NAME")
	if secretId == "" {
		secretId = "ts-acl-hostname-updater"
	}

	input := &secretsmanager.GetSecretValueInput{
		SecretId:     aws.String(secretId),
		VersionStage: aws.String("AWSCURRENT"),
	}

	secretsManager := secretsmanager.New(
		session.New(),
		aws.NewConfig().WithRegion(region),
	)

	result, err := secretsManager.GetSecretValue(input)
	if err != nil {
		return "", err
	}

	secret := []byte{}
	if result.SecretString != nil {
		secret = []byte(*result.SecretString)
	} else {
		secret, err = base64.StdEncoding.DecodeString(string(result.SecretBinary))
		if err != nil {
			return "", err
		}
	}

	values := make(map[string]string)
	err = json.Unmarshal(secret, &values)
	if err != nil {
		return "", err
	}

	apiKey, ok := values["API_KEY"]
	if !ok {
		return "", errors.New("No API_KEY in SecretsManager")
	}

	return apiKey, nil
}
